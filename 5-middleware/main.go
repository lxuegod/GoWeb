package main

import (
	"example/gee"
	"log"
	"net/http"
	"time"
)

func onlyForV2() gee.HandlerFunc {
	return func(c *gee.Context) {
		//	启动计时器
		t := time.Now()
		//	如果出现错误
		c.Fail(500, "Internal Server Error")
		//	计算处理时间
		log.Printf("[%d] %s in %v for group v2", c.StatusCode, c.Req.URL, time.Since(t))
	}
}

func main() {
	r := gee.New()
	//	全局中间件
	r.Use(gee.Logger())
	r.GET("/", func(c *gee.Context) {
		c.HTML(http.StatusOK, "<h1>Index Page</h1>")
	})

	v2 := r.Group("/v2")
	//	v2分组中间件
	v2.Use(onlyForV2())
	{
		v2.GET("/hello/:name", func(c *gee.Context) {
			// expect /hello/geektutu
			c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
		})
	}

	r.Run(":9999")
}
