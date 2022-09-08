package singleflight

import "sync"

//	call 代表正在进行中，或已经结束的请求
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

//	Group 管理不同key的请求(call)
type Group struct {
	mu sync.Mutex //	保护m
	m  map[string]*call
}

func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		//	如果请求正在进行中，则等待
		c.wg.Wait()
		//	请求结束 返回结果
		return c.val, c.err
	}
	c := new(call)
	//	发起请求前加锁
	c.wg.Add(1)
	//	添加到g.m，表名key已经有对应的请求在处理
	g.m[key] = c
	g.mu.Unlock()

	//	调用fn。发起请求
	c.val, c.err = fn()
	//	请求结束
	c.wg.Done()

	g.mu.Lock()
	//	更新g.m
	delete(g.m, key)
	g.mu.Unlock()

	//	返回结果
	return c.val, c.err
}
