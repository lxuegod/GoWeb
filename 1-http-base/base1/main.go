package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	//	设置路由
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/hello", helloHandler)
	//	启动Web服务  第一个参数 地址  第二个参数 处理所有的HTTP请求实例  nil 代表使用标准库中的实例处理
	log.Fatal(http.ListenAndServe(":9999", nil))
}

//	handler 回应 r.URL.Path
func indexHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "URL.Path = %q\n", req.URL.Path)
}

//	handler 回应 r.URL.Header
func helloHandler(w http.ResponseWriter, req *http.Request) {
	for k, v := range req.Header {
		fmt.Fprintf(w, "Header[%q] = %q\n", k, v)
	}
}
