package gin

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type H map[string]interface{}

/*
在 HandlerFunc 中，希望能够访问到解析的参数，
因此，需要对 Context 对象增加一个属性和方法，来提供对路由参数的访问。
将解析后的参数存储到Params中，通过c.Param("lang")的方式获取到对应的值。
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
}

// Context 构造器
func NewContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
	}
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

func (c *Context) HTML(code int, html string) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	c.Writer.Write([]byte(html))
}

func (c *Context) Param(key string) string {
	return c.Params[key]
}
