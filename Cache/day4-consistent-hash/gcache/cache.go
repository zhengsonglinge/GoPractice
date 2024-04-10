package gcache

import (
	"lru"
	"sync"
)

// 使用 sync.Mutex 封装 LRU 的几个方法，使之支持并发的读写。
type cache struct {
	mu         sync.Mutex
	lru        *lru.Cache
	cacheBytes int64
}

// mutex 锁住 lru 资源的访问
// 只有当 lru 不存在的时候才初始化，延迟初始化(Lazy Initialization)，提高性能，减少内存要求
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

// mutex 锁住 lru 资源的访问
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}
	return
}
