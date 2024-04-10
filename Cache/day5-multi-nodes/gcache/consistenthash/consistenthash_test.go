package consistenthash

import (
	"strconv"
	"testing"
)

func TestHashing(t *testing.T) {
	hash := New(3, func(data []byte) uint32 {
		i, _ := strconv.Atoi(string(data))
		return uint32(i)
	})

	// 哈希环中的排列如下，是 0-2 的字符串加上节点字符串得到的虚拟节点的 hash 值
	// 2,4,6,12,14,16,22,24,26
	// 其中 2，12，22 对应节点 "2"
	// 4，14，24 对应节点 "4"
	// 6,16,26 对应节点 "6"
	hash.Add("6", "4", "2")

	testCases := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}

	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}

	// 添加节点
	// 2,4,6,8,12,14,16,18,22,24,26,28
	hash.Add("8")

	testCases["27"] = "8"

	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}
}
