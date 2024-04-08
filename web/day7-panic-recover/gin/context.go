package gin

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type H map[string]interface{}

/*
请求的所有信息都保存在Context中，中间件也不例外。
接收到请求后，应查找所有应作用于该路由的中间件，保存在Context中，依次进行调用。
为什么依次调用后，还需要在Context中保存呢？
因为在设计中，中间件不仅作用在处理流程前，也可以作用在处理流程后，
即在用户定义的 Handler 处理完毕后，还可以执行剩下的操作。

在 Context 上下文中添加了 handlers 数组，用来存储中间件和用户方法
中间件和用户方法都是 HandlerFunc, 即 func(c *Context)
其中数组先存储中间件，再存储用户方法

例如：

	func A(c *Context) {
	    part1
	    c.Next()
	    part2
	}

	func B(c *Context) {
	    part3
	    c.Next()
	    part4
	}

中间件 A 和 B，和路由映射的 Handler。
c.handlers是这样的[A, B, Handler]
调用顺序：part1 -> part3 -> Handler -> part 4 -> part2
其实就是递归调用
*/
type Context struct {
	// 原始对象
	Writer http.ResponseWriter
	Req    *http.Request

	// 请求信息
	Path   string
	Method string
	Params map[string]string

	// 响应信息
	StatusCode int

	// middleware 中间件
	handlers []HandlerFunc
	index    int

	// 通过 engine 访问 HTML 模板
	engine *Engine
}

// Context 构造器
func NewContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
		index:  -1,
	}
}

// 逐个调用 handlers 中的中间件和用户方法
func (c *Context) Next() {
	c.index++
	s := len(c.handlers)
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

func (c *Context) Fail(code int, err string) {
	c.index = len(c.handlers)
	c.JSON(code, H{"message": err})
}

// 从表单中获取数据
func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

// 从请求 url 参数中获取值
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

// 设置响应的状态码
func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

// 设置响应头的键值对
func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

// 设置字符串格式的响应体
func (c *Context) String(code int, format string, values ...interface{}) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

// 设置 json 格式的响应体
func (c *Context) JSON(code int, obj ...interface{}) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	// NewEncoder 函数返回一个写入 c.Writer 的编码器
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

// 直接写入响应体
func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Writer.Write(data)
}

func (c *Context) HTML(code int, name string, data interface{}) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	if err := c.engine.htmlTemplates.ExecuteTemplate(c.Writer, name, data); err != nil {
		c.Fail(500, err.Error())
	}
}

func (c *Context) Param(key string) string {
	return c.Params[key]
}
