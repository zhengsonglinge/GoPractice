package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// 重命名 Hash 函数
// fn 采取依赖注入的方式，允许用于替换成自定义的 Hash 函数，也方便测试时替换
// 默认为 crc32.ChecksumIEEE 算法。
type Hash func(data []byte) uint32

type Map struct {
	hash     Hash           // Hash 函数
	replicas int            // 虚拟节点倍数，即每个真实节点对应的虚拟节点个数
	keys     []int          // 哈希环
	hashMap  map[int]string // 虚拟节点与真实节点的映射表，key 是虚拟节点哈希值，value 是真实节点名称
}

func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 添加真实节点
// 允许传入 0 或 多个真实节点的名称
func (m *Map) Add(realNode ...string) {
	for _, node := range realNode {
		// 每个真实节点对应 replicas 个虚拟节点
		for i := 0; i < m.replicas; i++ {
			// 虚拟节点的编号是 i + key
			// m.hash 计算虚拟节点的哈希值
			hash := int(m.hash([]byte(strconv.Itoa(i) + node)))
			// 将虚拟节点添加到环上
			m.keys = append(m.keys, hash)
			// 添加虚拟节点和真实节点的映射
			m.hashMap[hash] = node
		}
	}
	// 环上的哈希值排序，排序是为了二分查找
	sort.Ints(m.keys)
}

// 传入 key 值获取真实节点的名称
func (m *Map) Get(key string) string {
	// 没添加节点，哈希环上没有节点
	if len(m.keys) == 0 {
		return ""
	}

	// 计算 key 的哈希值
	hash := int(m.hash([]byte(key)))
	// 因为 m.keys 哈希环是有序的，因此可以用二分查找第一个大于等于 hash 的下标
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	// 返回真实节点的名称
	// 环形结构，需要取余
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
