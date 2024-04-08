package main

import (
	"fmt"
	"gin"
	"net/http"
	"text/template"
	"time"
)

/*
前后端分离：
后端提供 RESTful 接口，返回结构化的数据(通常为 JSON 或者 XML)。
前端使用 AJAX 技术请求到所需的数据，利用 JavaScript 进行渲染。

后端只关注于数据，接口返回值是结构化的，与前端解耦。
同一套后端服务能够同时支撑小程序、移动APP、PC端 Web 页面，以及对外提供的接口。

前后分离的一大问题在于，页面是在客户端渲染的，比如浏览器，这对于爬虫并不友好。
Google 爬虫已经能够爬取渲染后的网页，但是短期内爬取服务端直接渲染的 HTML 页面仍是主流。
*/

type student struct {
	Name string
	Age  int8
}

func FormatAsDate(t time.Time) string {
	year, month, day := t.Date()
	return fmt.Sprintf("%d-%02d-%02d", year, month, day)
}

func main() {
	r := gin.New()

	// 使用 gin.Logger 作为全局中间件
	r.Use(gin.Logger())

	r.SetFuncMap(template.FuncMap{"FormatAsDate": FormatAsDate})
	r.LoadHTMLGlob("templates/*")
	r.Static("assets", "./static")

	stu1 := &student{Name: "Tom", Age: 18}
	stu2 := &student{Name: "Jarry", Age: 19}

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "css.tmpl", nil)
	})
	r.GET("/student", func(c *gin.Context) {
		c.HTML(http.StatusOK, "arr.tmpl", gin.H{
			"title":  "gin",
			"stuArr": [2]*student{stu1, stu2},
		})
	})
	r.GET("/date", func(c *gin.Context) {
		c.HTML(http.StatusOK, "custom_func.tmpl", gin.H{
			"title": "gin",
			"now":   time.Date(2024, 01, 01, 0, 0, 0, 0, time.UTC),
		})
	})

	r.Run(":9999")
}
