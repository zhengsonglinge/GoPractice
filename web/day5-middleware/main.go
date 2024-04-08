package main

import (
	"gin"
	"log"
	"net/http"
	"time"
)

/*
中间件(middlewares)，就是非业务的技术类组件。
框架需要有一个插口，允许用户自己定义功能，嵌入到框架中，仿佛这个功能是框架原生支持的一样。

gin 的中间件的定义与路由映射的 Handler 一致，处理的输入是Context对象。
插入点是框架接收到请求初始化Context对象后，允许用户使用自己定义的中间件做一些额外的处理，例如记录日志等， 以及对Context进行二次加工。
另外通过调用(*Context).Next()函数，中间件可等待用户自己定义的 Handler处理结束后，再做一些额外的操作，例如计算本次处理所用时间等。
即 gin 的中间件支持用户在请求被处理的前后，做一些额外的操作。
*/
// 自定义一个中间件，中间件也是一个以 Context 为参数的 gin.HandlerFunc
func onlyForv2() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()
		c.Fail(500, "Internal Server Error")
		log.Printf("[%d] %s in %v for group v2", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}
func main() {
	r := gin.New()

	// 使用 gin.Logger 作为全局中间件
	r.Use(gin.Logger())
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "<h1>Hello Web<h1>")
	})

	v2 := r.Group("/v2")
	// v2 路由组中使用 onlyForv2 中间件
	v2.Use(onlyForv2())
	{
		v2.GET("/hello/:name", func(c *gin.Context) {
			c.String(http.StatusOK, "hello %s,you're at %s\n", c.Param("name"), c.Path)
		})
	}

	r.Run(":9999")
}

/*
curl 127.0.0.1:9999/
<h1>Hello Web<h1>
>>>log
2024/04/08 03:27:17 [200] / in 0s

curl 127.0.0.1:9999/v2/hello
[{"message":"Internal Server Error"}]
>>>log
2024/04/08 03:28:42 [500] /v2/hello in 0s for group v2
2024/04/08 03:28:42 [500] /v2/hello in 901.6µs
*/
