package main

import (
	"fmt"
	"gin"
	"net/http"
)

/*
设计 gin 框架，使用 Engine 代理所有的 HTTP 请求
添加 GET、POST 请求的处理方式
*/
func main() {
	r := gin.New()
	r.GET("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "URL.Path = %q\n", req.URL.Path)
	})
	r.GET("/hello", func(w http.ResponseWriter, req *http.Request) {
		for k, v := range req.Header {
			fmt.Fprintf(w, "Header[%q] = %q\n", k, v)
		}
	})

	r.Run(":9999")
}

/*
curl 127.0.0.1:9999/
URL.Path = "/"

curl 127.0.0.1:9999/hello
Header["User-Agent"] = ["curl/8.4.0"]
Header["Accept"] = [""]
*/
