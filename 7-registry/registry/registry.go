//	GoRegistry 是一个简单的注册中心，提供以下功能
//	增加一个server并接收heartbeat保持sever活着
//	返回sever列表，同时删除死掉的sever
package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type GoRegistry struct {
	timeout time.Duration
	mu      sync.Mutex
	servers map[string]*ServerItem
}

type ServerItem struct {
	Addr  string
	start time.Time
}

const (
	//	默认路径
	defaultPath = "/_gorpc_/registry"
	//	默认超时时间 5 分钟
	defaultTimeout = time.Minute * 5
)

//	New 创建一个有超时设置的注册实例
func New(timeout time.Duration) *GoRegistry {
	return &GoRegistry{
		servers: make(map[string]*ServerItem),
		timeout: timeout,
	}
}

var DefaultGoRegister = New(defaultTimeout)

//	添加服务实例
func (r *GoRegistry) putServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.servers[addr]
	if s == nil { //	不存在创建服务
		r.servers[addr] = &ServerItem{Addr: addr, start: time.Now()}
	} else { //	如果存在更新start
		s.start = time.Now()
	}
}

//	返回可用的服务列表
func (r *GoRegistry) aliveServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var alive []string
	for addr, s := range r.servers {
		if r.timeout == 0 || s.start.Add(r.timeout).After(time.Now()) {
			alive = append(alive, addr)
		} else { //	存在超时服务 删除
			delete(r.servers, addr)
		}
	}
	sort.Strings(alive)
	return alive
}

//	Runs at /_gorpc_/registry 注册中心信息采用HTTP提供服务
func (r *GoRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	//	返回可用服务列表
	case "GET":
		w.Header().Set("X-Gorpc-Servers", strings.Join(r.aliveServers(), ","))
	case "POST":
		//	添加服务实例/添加心跳
		addr := req.Header.Get("X-Gorpc-Server")
		if addr == "" {
			//	500
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		//	405
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

//	HandleHTTP 注册HTTP处理程序
func (r *GoRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, r)
	log.Println("rpc registry path:", registryPath)
}

func HandleHTTP() {
	DefaultGoRegister.HandleHTTP(defaultPath)
}

//	Heartbeat 定时向服务中心发送心跳
func Heartbeat(registry, addr string, duration time.Duration) {
	if duration == 0 {
		//	发送心跳周期默认比 注册中心过期时间少1min
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}
	var err error
	err = sendHeartbeat(registry, addr)
	//	定时器
	go func() {
		t := time.NewTicker(duration)
		for err == nil {
			<-t.C
			err = sendHeartbeat(registry, addr)
		}
	}()
}

func sendHeartbeat(registry, addr string) error {
	log.Println(addr, "send heart beat to registry", registry)
	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("X-Gorpc-Server", addr)

	if _, err := httpClient.Do(req); err != nil {
		log.Println("rpc server: heart beat err: ", err)
		return err
	}
	return nil
}
