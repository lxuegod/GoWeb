package gorpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"gorpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)

const Number = 0x1a2b3c

type Option struct {
	//	标记该请求为rpc请求
	Number int
	//	编解码方式 string类型
	CodecType codec.Type
	//	连接超时 默认10s
	ConnectTimeout time.Duration
	//	处理请求超时 默认0 表示不设限
	HandleTimeout time.Duration
}

//	DefaultOption 默认选择为GobType
var DefaultOption = &Option{
	Number:         Number,
	CodecType:      codec.GobType,
	ConnectTimeout: time.Second * 10,
}

//	Server 一次rpc服务
type Server struct {
	serviceMap sync.Map
}

//	NewServer 构造函数
func NewServer() *Server {
	return &Server{}
}

// ServeConn 处理一次rpc连接下的请求 直到客户端断开请求
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() { _ = conn.Close() }()
	var opt Option
	//	反序列化得到 Option 实例
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	//	检查 Number 值
	if opt.Number != Number {
		log.Printf("rpc server: invalid codec type #{opt.CodecType}")
		return
	}
	//	检查编码格式
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server: invalid codec type #{opt.CodecType}")
		return
	}

	server.serverCodec(f(conn), &opt)
}

//	serveCodec 编解码处理
func (server *Server) serverCodec(cc codec.Codec, opt *Option) {
	//	互斥锁 确保一个 response 完整地发出
	sending := new(sync.Mutex)
	//	用于同步 等到所有请求处理完
	wg := new(sync.WaitGroup)

	for {
		//	1.读取请求
		req, err := server.readRequest(cc)
		if err != nil {
			if req != nil {
				//	请求无法恢复 直接断开连接
				break
			}
			req.h.Error = err.Error()
			//	3.回复请求
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		//	2.处理请求 计数器 +1
		wg.Add(1)
		go server.handleRequest(cc, req, sending, wg, opt.HandleTimeout)
	}
	//	阻塞 直到请求处理完
	wg.Wait()
	_ = cc.Close()
}

//	request 存储 请求信息
type request struct {
	//	请求头
	h *codec.Header
	//	请求参数
	argv reflect.Value
	//	回复参数
	replyv reflect.Value
	mtype  *methodType
	svc    *service
}

//	readRequestHeader 读取请求头
func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error: ", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) findServer(serviceMethod string) (svc *service, mtype *methodType, err error) {
	//	检查请求服务模式
	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc service: service/method request ill-formed: " + serviceMethod)
		return
	}
	//	根据dot划分 服务名.方法名
	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:]

	//	svci -> 找到对应的Service实例
	svci, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc server: can`t find service " + serviceName)
		return
	}
	//	在对应 Service实例中 找到对应 methodType
	svc = svci.(*service)
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("rpc server: can`t find method " + methodName)
	}
	return
}

// readRequest 读取请求
func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	////TODO: 还未知请求的 argv 参数
	////先当作 string
	//req.argv = reflect.New(reflect.TypeOf(""))
	//if err = cc.ReadBody(req.argv.Interface()); err != nil {
	//	log.Println("rpc server: read argv err:", err)
	//}
	req.svc, req.mtype, err = server.findServer(h.ServiceMethod)
	if err != nil {
		return req, err
	}
	//	创建入参实例
	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()

	//	注意argvi的值类型为指针或值类型
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}
	if err = cc.ReadBody(argvi); err != nil {
		log.Println("rpc server: read body err:", err)
		return req, err
	}

	return req, nil
}

//	sendResponse 发送响应
func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	//	这里上锁 保证相应的有序发出	防止其他的 goroutine 也在同一个缓冲区写入
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response")
	}
}

//	handleRequest 处理请求
func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup, timeout time.Duration) {
	defer wg.Done()

	//	一次处理 分为两个过程
	//	用于事件通信
	//	TODO 可以设置为 缓存信道 防止timeout后协程阻塞无法关闭 造成的内存泄露
	called := make(chan struct{})
	sent := make(chan struct{})

	go func() {
		err := req.svc.call(req.mtype, req.argv, req.replyv)

		called <- struct{}{}
		if err != nil {
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
			sent <- struct{}{}
			return
		}
		server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
		sent <- struct{}{}
	}()

	if timeout == 0 {
		<-called
		<-sent
		return
	}
	select {
	case <-called:
		<-sent
	case <-time.After(timeout):
		req.h.Error = fmt.Sprintf("rpc server: request handle timeout: expect within #{timeout}")
		server.sendResponse(cc, req.h, invalidRequest, sending)
		//	如果为缓存信道，则可以将下面注释掉
		<-called
		<-sent
	}
}

//	invalidRequest 发生错误的时候 argv 占位符
var invalidRequest = struct{}{}

//	DefaultServer *Server 的默认实例
var DefaultServer = NewServer()

//	Accept 接受Sever请求
func (server *Server) Accept(lis net.Listener) {
	//	循环等待 socket 连接建立
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error ", err)
			return
		}

		//	开启子协程 处理连接请求
		go server.ServeConn(conn)

	}
}

//	Accept 包装Accept函数 方便使用
//	一次服务启动如下
//	lis, _ := net.Listen("tcp", ":9999")
//	gorpc.Accept(lis)
func Accept(lis net.Listener) { DefaultServer.Accept(lis) }

//	Register 在服务器中注册
func (server *Server) Register(rcvr interface{}) error {
	s := newService(rcvr)
	//	.LoadOrStore -> getOfDefault(Java map)
	if _, dup := server.serviceMap.LoadOrStore(s.name, s); dup {
		return errors.New("rpc: service already defined: " + s.name)
	}
	return nil
}

// Register 以 DefaultServer 注册
func Register(rcvr interface{}) error {
	return DefaultServer.Register(rcvr)
}
