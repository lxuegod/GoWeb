package gee

import (
	"fmt"
	"net/http"
)

//	HandleFunc 被gee定义的请求处理程序
type HandleFunc func(w http.ResponseWriter, r *http.Request)

//	Engine 实现ServeHTTP接口
type Engine struct {
	router map[string]HandleFunc
}

//	New 是gee.Engine的构造函数
func New() *Engine {
	return &Engine{router: make(map[string]HandleFunc)}
}

func (engine *Engine) addRoute(method, pattern string, handler HandleFunc) {
	key := method + "-" + pattern
	engine.router[key] = handler
}

//	GET 定义增加get请求的方法
func (engine *Engine) GET(pattern string, handler HandleFunc) {
	engine.addRoute("GET", pattern, handler)
}

//	POST 定义增加post请求的方法
func (engine *Engine) POST(pattern string, handler HandleFunc) {
	engine.addRoute("POST", pattern, handler)
}

//	Run ListenAndServe的封装
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	key := req.Method + "-" + req.URL.Path
	if handler, ok := engine.router[key]; ok {
		handler(w, req)
	} else {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "404 NOT FOUND: %s\n", req.URL)
	}
}
