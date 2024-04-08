package main

import (
	"gin"
	"net/http"
)

/*
Go 语言中，比较常见的错误处理方法是返回 error，由调用者决定后续如何处理。
但是如果是无法恢复的错误，可以手动触发 panic，当然如果在程序运行过程中出现了类似于数组越界的错误，panic 也会被触发。
panic 会中止当前执行的程序，退出。

panic 会导致程序被中止，但是在退出前，会先处理完当前协程上已经defer 的任务，执行完成后再退出。

Go 语言还提供了 recover 函数，可以避免因为 panic 发生而导致整个程序终止，recover 函数只在 defer 中生效。
*/

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello\n")
	})
	r.GET("/panic", func(c *gin.Context) {
		names := []string{"name"}
		c.String(http.StatusOK, names[100])
	})

	r.Run(":9999")
}

/*
curl 127.0.0.1:9999/panic
[{"message":"InternalServerError"}]

untime error: index out of range [100] with length 1
Traceback:
        D:/go1.21/src/runtime/panic.go:914
        D:/go1.21/src/runtime/panic.go:114
        D:/GoPractice/web/day7-panic-recover/main.go:26
        D:/GoPractice/web/day7-panic-recover/gin/context.go:77
        D:/GoPractice/web/day7-panic-recover/gin/recover.go:42
        D:/GoPractice/web/day7-panic-recover/gin/context.go:77
        D:/GoPractice/web/day7-panic-recover/gin/logger.go:14
        D:/GoPractice/web/day7-panic-recover/gin/context.go:77
        D:/GoPractice/web/day7-panic-recover/gin/router.go:103
        D:/GoPractice/web/day7-panic-recover/gin/gin.go:68
        D:/go1.21/src/net/http/server.go:2939
        D:/go1.21/src/net/http/server.go:2010
        D:/go1.21/src/runtime/asm_amd64.s:1651

2024/04/09 02:26:34 [500] /panic in 12.5911ms
*/
