package gorpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"gorpc/codec"
	"io"
	"log"
	"net"
	"sync"
)

//	Call 一次rpc调用所需要的信息
type Call struct {
	//	序列号
	Seq uint64
	//	请求方法
	ServiceMethod string
	//	请求参数
	Args interface{}
	//	方法的返回值
	Reply interface{}
	//	错误信息
	Error error
	//	调用后的回调
	Done chan *Call
}

func (call *Call) done() {
	call.Done <- call
}

type Client struct {
	//	消息编/解码器
	cc codec.Codec
	//	发起连接前的确认(请求类型/编码方式)
	opt *Option
	//	保证Client并发时可用性
	sending sync.Mutex
	//	每个请求的消息头
	header codec.Header
	//	保证内部服务的有序性
	mu sync.Mutex
	//	发送请求的编号
	seq uint64
	//	存储未发送完的请求 k:v -> 请求:编号实例
	pending map[uint64]*Call
	//	是否关闭rpc请求(用户正常关闭)
	closing bool
	//	服务停止(用于非正常停止)
	shutdown bool
}

var _ io.Closer = (*Client)(nil)

var ErrShutdown = errors.New("connection is shut down ")

//	Close关闭连接
func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing {
		return ErrShutdown
	}
	client.closing = true
	return client.cc.Close()
}

//	IsAvailable 确保client服务正常前提
func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.shutdown && !client.closing
}

//	register 客户端注册rpc请求
func (client *Client) register(call *Call) (uint64, error) {
	//	方法内部上锁 防止并发问题 -> 该方法被被其他client调用
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing || client.shutdown {
		return 0, ErrShutdown
	}
	call.Seq = client.seq
	client.pending[call.Seq] = call
	//	序号++
	client.seq++
	return call.Seq, nil
}

//	removeCall 客户端移除rpc请求
func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

//	terminateCalls rpc请求错误
//	defer 处理顺序: client.mu.Unlock -> client.sending.Unlock
func (client *Client) terminateCalls(err error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	client.shutdown = true
	//	将所有错误信息通知等待处理中的call
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}

//	send 发送请求
func (client *Client) send(call *Call) {
	//	加锁确保请求信息发送完整
	client.sending.Lock()
	defer client.sending.Unlock()

	//	先注册前请求信息
	seq, err := client.register(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}

	//	准备请求头
	client.header.ServiceMethod = call.ServiceMethod
	client.header.Seq = seq
	client.header.Error = ""

	//	编码 发送请求
	if err := client.cc.Write(&client.header, call.Args); err != nil {
		call := client.removeCall(seq)
		//	call 可能为 nil，通常是Write部分失败
		//	client 已经收到回复并处理
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

//	Go 对外暴露给用户的RPC调用接口
//	异步接口 返回Call实例
func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		//	done 没有缓冲区
		log.Panic("rpc client: done channel is unbuffered")
	}
	//	构造一个Call请求
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	//	发送请求
	//	TODO 此处的done是同步等待
	client.send(call)
	return call
}

//	Call 封装Go
//	同步接口 call.Done，等待响应返回
func (client *Client) Call(serviceMethod string, args, reply interface{}) error {
	//	TODO chan数量为1 保证同步
	call := <-client.Go(serviceMethod, args, reply, make(chan *Call, 1)).Done
	return call.Error
}

//	receive 接收响应
func (client *Client) receive() {
	var err error
	for err == nil {
		var h codec.Header
		if err = client.cc.ReadHeader(&h); err != nil {
			break
		}
		call := client.removeCall(h.Seq)
		switch {
		case call == nil:
			//	TODO call不存在 可能是请求没有发送完整，或者因为其他原因被取消，但是服务端仍然处理了？
			err = client.cc.ReadBody(nil)
		case h.Error != "":
			//	call存在 但是服务器处理出错
			call.Error = fmt.Errorf(h.Error)
			err = client.cc.ReadBody(nil)
			call.done()
		default:
			//	服务器处理正常
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
	client.terminateCalls(err)
}

//	NewClient 创建一个客户端实例
func NewClient(conn net.Conn, opt *Option) (*Client, error) {
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		err := fmt.Errorf("invalid codec type %s", opt.CodecType)
		log.Println("rpc client: codec error:", err)
		return nil, err
	}
	//	发送option 编码给服务器
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc client: options error :", err)
		_ = conn.Close()
		return nil, err
	}
	return NewClientCodec(f(conn), opt), nil
}

func NewClientCodec(cc codec.Codec, opt *Option) *Client {
	client := &Client{
		seq:     1, //	seq从1开始，0意味着无效的序列号
		cc:      cc,
		opt:     opt,
		pending: make(map[uint64]*Call),
	}
	//	开启一个协程 receive 响应
	go client.receive()
	return client
}

//	parseOptions 验证options编码信息
func parseOptions(opts ...*Option) (*Option, error) {
	//	用户输入的Options有错误时
	//	返回默认的DefaultOption
	if len(opts) == 0 || opts[0] == nil {
		return DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("number of options more than 1")
	}
	opt := opts[0]
	opt.Number = DefaultOption.Number
	if opt.CodecType == "" {
		opt.CodecType = DefaultOption.CodecType
	}
	return opt, nil
}

//	Dail 传入服务端地址
func Dail(network, address string, opts ...*Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	//	关闭连接
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	return NewClient(conn, opt)
}
