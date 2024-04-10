package gcache

// 抽象了一个只读数据结构 ByteView 用来表示缓存值
type ByteView struct {
	b []byte // 存储缓存数据，使用 byte 数组可以支持不同的类型
}

// 实现 Len 方法可以实现 lru.Value 接口
func (v ByteView) Len() int {
	return len(v.b)
}

// b 是只读的，返回一个拷贝，防止缓存值被外部程序修改
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}
func (v ByteView) String() string {
	return string(v.b)
}
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
