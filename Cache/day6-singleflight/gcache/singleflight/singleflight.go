package singleflight

import "sync"

/*
假设对数据库的访问没有做任何限制的，很可能向数据库也发起 N 次请求，容易导致缓存击穿和穿透。
即使对数据库做了防护，HTTP 请求是非常耗费资源的操作
针对相同的 key，使用 singleflight 将所有请求合并成一次请求
*/

// 正在进行中或已经结束的请求
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// 管理不同 key 的请求 （call）
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

/*
Do 方法，接收 2 个参数，第一个参数是 key，第二个参数是一个函数 fn。
Do 的作用就是，针对相同的 key，无论 Do 被调用多少次，函数 fn 都只会被调用一次，等待 fn 调用结束了，返回返回值或错误
*/
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		// 等待 c.wg.Done 函数，即 fn 函数执行完毕
		c.wg.Wait()
		return c.val, c.err
	}

	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
