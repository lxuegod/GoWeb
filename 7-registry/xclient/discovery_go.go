package xclient

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type GoRegistryDiscovery struct {
	*MultiServerDiscovery
	//	注册中心的地址
	registry string
	//	服务列表的超时时间
	timeout time.Duration
	//	最后从注册中心更新服务列表的时间
	lastUpdate time.Time
}

//	默认10s过期
const defaultUpdateTimeout = time.Second * 10

//	NewGoRegistryDiscovery 初始化
func NewGoRegistryDiscovery(registryAddr string, timeout time.Duration) *GoRegistryDiscovery {
	if timeout == 0 {
		timeout = defaultUpdateTimeout
	}

	d := &GoRegistryDiscovery{
		MultiServerDiscovery: NewMultiServerDiscovery(make([]string, 0)),
		registry:             registryAddr,
		timeout:              timeout,
	}
	return d
}

//	实现Update方法 根据入参更新 服务列表
func (d *GoRegistryDiscovery) Update(servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = servers
	d.lastUpdate = time.Now()
	return nil
}

//	实现Refresh方法 超时 自动更新服务列表
func (d *GoRegistryDiscovery) Refresh() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.lastUpdate.Add(d.timeout).After(time.Now()) {
		return nil
	}
	log.Println("rpc registry: refresh servers from registry", d.registry)
	resq, err := http.Get(d.registry)
	if err != nil {
		log.Println("rpc registry refresh err: ", err)
		return err
	}
	//	返回可用服务列表
	servers := strings.Split(resq.Header.Get("X-Gorpc-Servers"), ",")
	d.servers = make([]string, 0, len(servers))
	for _, server := range servers {
		if strings.TrimSpace(server) != "" {
			d.servers = append(d.servers, strings.TrimSpace(server))
		}
	}
	d.lastUpdate = time.Now()
	return nil
}

//	Get 根据负载均衡模式 返回一个可用服务实例
func (d *GoRegistryDiscovery) Get(mode SelectMode) (string, error) {
	if err := d.Refresh(); err != nil {
		return "", err
	}
	return d.MultiServerDiscovery.Get(mode)
}

//	返回全部服务实例
func (d *GoRegistryDiscovery) GetAll() ([]string, error) {
	//	需要先确保服务裂变没有过期
	if err := d.Refresh(); err != nil {
		return nil, err
	}
	return d.MultiServerDiscovery.GetAll()
}
