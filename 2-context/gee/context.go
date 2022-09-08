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
	//	响应信息
	StatusCode int
}

//	newContext 构造Context
func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
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
func (c *Context) HTML(code int, html string) {
	c.SetHeader("Contend-Type", "text/html")
	c.Status(code)
	c.Writer.Write([]byte(html))
}
