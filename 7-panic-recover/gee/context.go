package gee

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type H map[string]interface{}

type Context struct {
	//	origin objects
	Writer http.ResponseWriter
	Req    *http.Request
	//	请求信息
	Path   string
	Method string
	Params map[string]string
	//	响应信息
	StatusCode int
	//	中间件
	handlers []HandlerFunc
	index    int
	//	engine 指针
	engine *Engine //	通过Context访问Engine中的HTML模板
}

//	Param 获取对应的值
func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

//	newContext 构造Context
func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
		index:  -1,
	}
}

//	PostForm 访问PostForm参数
func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

//	Query 访问Query参数
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

func (c *Context) SetHeader(key, value string) {
	c.Writer.Header().Set(key, value)
}

//	String 快速构造string响应
func (c *Context) String(code int, format string, values ...interface{}) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

//	JSON 快速构造json响应
func (c *Context) JSON(code int, obj interface{}) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		//	500
		http.Error(c.Writer, err.Error(), 500)
	}
}

//	Data 快速构造data响应
func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Writer.Write(data)
}

//	HTML 快速构造html响应
func (c *Context) HTML(code int, name string, data interface{}) {
	c.SetHeader("Contend-Type", "text/html")
	c.Status(code)
	if err := c.engine.htmlTemplates.ExecuteTemplate(c.Writer, name, data); err != nil {
		c.Fail(500, err.Error())
	}
}

//	Next等待执行其他的中间件或用户的Handler
func (c *Context) Next() {
	c.index++
	s := len(c.handlers)
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}
