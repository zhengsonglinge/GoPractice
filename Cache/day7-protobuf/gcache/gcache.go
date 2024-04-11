package gcache

import (
	"fmt"
	"gcache/gcachepb"
	"gcache/singleflight"
	"log"
	"sync"
)

/*
Getter 的作用：
如果缓存不存在，应从数据源（文件，数据库等）获取数据并添加到缓存中。
设计了一个回调函数(callback)，在缓存不存在时，调用这个函数，得到源数据。
用户来定义这个回调函数，即用户处理源数据从哪来
*/
type Getter interface {
	Get(key string) ([]byte, error)
}

// 函数类型，实现了 Getter
// 函数类型实现接口，称为接口型函数
// 使用者在调用时既能传入函数作为参数，也能传入实现了该接口的结构体作为参数
type GetterFunc func(key string) ([]byte, error)

// 回调函数，实现了接口 Getter 的方法
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// 最重要的数据结构
// 一个 Group 可以认为是一个缓存的命名空间
type Group struct {
	name      string              // 每个 Group 拥有一个唯一的名称 name
	getter    Getter              // 缓存未命中时获取源数据的回调(callback)
	mainCache cache               // 并发缓存
	peers     PeerPicker          // 远程节点选择器
	loader    *singleflight.Group // 合并请求，避免缓存穿透
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// 注册一个 PeerPicker 节点选择器用来选择远程节点
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// Get 函数用来查找缓存
// 缓存存在直接返回
// 不存在调用 Getter 接口的 Get 方法从源数据获取数据并返回
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GCache] hit")
		return v, nil
	}

	return g.load(key)
}

// 缓存未命中，选择加载数据
// 先选择远程节点获取数据，如果远程节点数据获取失败则调用本地获取数据
func (g *Group) load(key string) (value ByteView, err error) {
	// 使用 singleflight 合并请求
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err := g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GCache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

// 使用 PeerGetter 的 Get 方法从远程节点获取数据
// 使用 protobuf 代替原来的 Get 函数
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &gcachepb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &gcachepb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}

// 从本地获获取源数据
// 调用 Getter 的 Get 函数获取源数据
// 将获取到的数据同时加载到内存中
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

// 将数据加载到内存
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
