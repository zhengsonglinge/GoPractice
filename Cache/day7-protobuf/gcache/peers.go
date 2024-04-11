package gcache

import "gcache/gcachepb"

// 节点选择器
type PeerPicker interface {
	// 根据 key 选择相应节点 PeerGetter
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// 相当于 HTTP 客户端
type PeerGetter interface {
	// 用于从对应 group 查找缓存值
	// 用 protobuf 生成的代码代替,in 和 out 都是指针，不用返回 out 了
	Get(in *gcachepb.Request, out *gcachepb.Response) error
}
