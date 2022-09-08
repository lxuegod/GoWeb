package gee

import (
	"log"
	"net/http"
)

//	HandlerFunc 被gee定义的请求处理程序
type HandlerFunc func(*Context)

type RouterGroup struct {
	prefix      string
	middlewares []HandlerFunc //	支持 middleware
	parent      *RouterGroup  //	支持 nesting
	engine      *Engine       //	所有分组共享一个Engine实例  通过Engine间接地访问各种接口
}

//	Engine 实现ServeHTTP接口
//	Engine 拥有RouterGroup所有的能力
type Engine struct {
	*RouterGroup
	router *router
	groups []*RouterGroup //	存储所有的分组
}

//	New 是gee.Engine的构造函数
func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

//	Group 创造一个新的RouterGroup
//	所有的分组共享同一个Engine实例
func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		parent: group,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

func (group *RouterGroup) addRoute(method, comp string, handler HandlerFunc) {
	pattern := group.prefix + comp
	log.Printf("Route %4s - %s", method, pattern)
	group.engine.router.addRoute(method, pattern, handler)
}

//	GET 定义增加get请求的方法
func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute("GET", pattern, handler)
}

//	POST 定义增加post请求的方法
func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute("POST", pattern, handler)
}

//	Run ListenAndServe的封装
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := newContext(w, req)
	engine.router.handle(c)
}
