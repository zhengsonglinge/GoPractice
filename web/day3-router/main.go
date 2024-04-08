package main

import (
	"gin"
	"net/http"
)

/*
之前的路由表用 map 实现，只能实现静态路由
动态路由，即一条路由规则可以匹配某一类型而非某一条固定的路由。
例如/hello/:name，可以匹配/hello/tom、/hello/jarry等。

实现动态路由最常用的数据结构，被称为前缀树(Trie树)，前缀树每个子节点代表一个字符。
压缩前缀树是前缀树的一种优化形式，用于解决前缀树在存储大量短字符串时占用空间过大的问题。

HTTP请求的路径恰好是由/分隔的多段构成的，因此，每一段可以作为前缀树的一个节点。
*/
func main() {
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "<h1>Hello Web<h1>")
	})
	r.GET("/hello", func(c *gin.Context) {
		// expact /hello?name=testName
		c.String(http.StatusOK, "hello %s,you're at %s\n", c.Query("name"), c.Path)
	})
	r.GET("/hello/:name", func(c *gin.Context) {
		// expact /hello/testName
		c.String(http.StatusOK, "hello %s,you're at %s\n", c.Param("name"), c.Path)
	})
	r.GET("/assets/*filepath", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"filepath": c.Param("filepath")})
	})

	r.Run(":9999")
}

/*
curl 127.0.0.1:9999/hello/testName
hello testName,you're at /hello/testName

curl 127.0.0.1:9999/assets/css/file.js
[{"filepath":"css/file.js"}]
*/
