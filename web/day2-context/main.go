package main

import (
	"gin"
	"net/http"
)

/*
将路由(router)独立出来，方便之后增强。
设计上下文(Context)，封装 Request 和 Response ，提供对 JSON、HTML 等返回类型的支持。

Context 封装了 http.ResponseWriter 和 http.Request，减少了许多重复代码。

Context 随着每一个请求的出现而产生，请求的结束而销毁，和当前请求强相关的信息都应由 Context 承载。
设计 Context 结构，扩展性和复杂性留在了内部，而对外简化了接口。

例如：

	obj = map[string]interface{}{
	    "name": "test",
	    "password": "1234",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(obj); err != nil {
	    http.Error(w, err.Error(), 500)
	}

封装之后：

	c.JSON(http.StatusOK, gin.H{
	    "username": c.PostForm("username"),
	    "password": c.PostForm("password"),
	})
*/
func main() {
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "<h1>Hello Web<h1>")
	})
	r.GET("/hello", func(c *gin.Context) {
		c.String(http.StatusOK, "hello %s,you're at %s\n", c.Query("name"), c.Path)
	})

	r.POST("/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"username": c.PostForm("username"),
			"password": c.PostForm("password"),
		})
	})

	r.Run(":9999")
}

/*
curl 127.0.0.1:9999/
<h1>Hello Web<h1>

curl 127.0.0.1:9999/hello?name=test
hello test,you're at /hello

curl "http://127.0.0.1:9999/login" -X POST -d "username=name&password=password"
[{"password":"password","username":"name"}]

curl 127.0.0.1:9999/xxx
404 NOT FOUND: /xx
*/
