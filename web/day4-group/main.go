package main

import (
	"gin"
	"net/http"
)

/*
分组控制(Group Control)是 Web 框架应提供的基础功能之一。
分组控制即对路由进行分组，通过路由的前缀区分不同组。
并且支持分组的嵌套。例如/post是一个分组，/post/a和/post/b可以是该分组下的子分组。
作用在/post分组上的中间件(middleware)，也都会作用在子分组，子分组还可以应用自己特有的中间件。
中间件可以给框架提供无限的扩展能力，应用在分组上，可以使得分组控制的收益更为明显
*/
func main() {
	r := gin.New()
	r.GET("/index", func(c *gin.Context) {
		c.HTML(http.StatusOK, "<h1>Index Page</h1>")
	})
	v1 := r.Group("/v1")
	{
		v1.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, "<h1>Hello Web<h1>")
		})
		v1.GET("/hello", func(c *gin.Context) {
			// expact /hello?name=testName
			c.String(http.StatusOK, "hello %s,you're at %s\n", c.Query("name"), c.Path)
		})

	}
	v2 := r.Group("/v2")
	{
		v2.GET("/hello/:name", func(c *gin.Context) {
			// expact /hello/testName
			c.String(http.StatusOK, "hello %s,you're at %s\n", c.Param("name"), c.Path)
		})
		v2.POST("/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"username": c.PostForm("username"),
				"password": c.PostForm("password"),
			})
		})
	}

	r.Run(":9999")
}

/*
curl 127.0.0.1:9999/v1/hello?name=test
hello test,you're at /v1/hello

curl 127.0.0.1:9999/v2/hello/test
hello test,you're at /v2/hello/test
*/
