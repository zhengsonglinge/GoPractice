package lru

import "container/list"

/*
实现 lru 算法的核心是两个数据结构体
一个 map，一个双向链表
其中 map 的 key 是操作的值，value 是指向双向链表中节点的指针
map 的作用是快速找到调用的节点，将节点放到链表头
双向链表的作用是快速从链表末尾删除元素
*/

// LRU 缓存
type Cache struct {
	maxBytes  int64                         // 允许使用的最大内存
	nbytes    int64                         // 当前已使用的内存
	ll        *list.List                    // go 标准库 list.List
	cache     map[string]*list.Element      // list.Element 是双向边表中的节点
	OnEvicted func(key string, value Value) // 某条记录被移除时的回调函数，可以为 nil
}
type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int
}

// Cache 的构造器
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// 查找功能
// 从 map 中获取双向链表的节点，并将节点移到队头
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// 删除最近最久未使用元素
func (c *Cache) RemoveOldest() {
	// 获取队尾元素
	ele := c.ll.Back()
	if ele != nil {
		// 删除队尾元素
		c.ll.Remove(ele)
		// 删除 map 中的节点
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		// 当前使用内存减少更新
		c.nbytes -= int64(len(kv.key) + kv.value.Len())
		// 调用回调函数
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// 新增/修改
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		// 如果键存在则更新键，将节点移动到队头
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len() - kv.value.Len())
		kv.value = value
	} else {
		// 如果不存在则创建一个节点插入到队头，map 中插入键值对
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key) + value.Len())
	}
	// 如果当前内存大于最大内存则移除队尾元素
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// 获取当前有多少数据
func (c *Cache) Len() int {
	return c.ll.Len()
}
