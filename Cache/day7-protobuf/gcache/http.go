package gcache

import (
	"fmt"
	"gcache/consistenthash"
	"gcache/gcachepb"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"
)

const (
	defaultBasePath = "/_gcache/"
	defaultReplicas = 50
)

// 创建一个结构体 HTTPPool，作为承载节点间 HTTP 通信的核心数据结构
// HTTPPool 既具备了提供 HTTP 服务的能力，接收客户端请求
// 也具备据具体的 key，创建 HTTP 客户端从远程节点获取缓存值的能力
type HTTPPool struct {
	self     string // 基础 url，记录自己的地址 e.g. "https://example.net:8000"
	basePath string // 节点间通信地址的前缀，例如 http://example.com/_gcache/ 开头的请求就是用于节点间访问

	// 添加节点选择功能
	mu          sync.Mutex             // 为 peers 和 httpGetters 加锁
	peers       *consistenthash.Map    // 一致性哈希的虚拟节点和真实节点的映射
	httpGetters map[string]*httpGetter // 键值示例 "http://10.0.0.2:8008"，每个远程节点对应一个 httpGetter
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// 服务端功能，代理所有的 HTTP 请求
// 约定访问路径格式为 /<basepath>/<groupname>/<key>，通过 groupname 得到 group 实例，再使用 group.Get(key) 获取缓存数据。
// 最终使用 w.Write() 将缓存值作为 httpResponse 的 body 返回。
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	p.Log("%s %s", r.Method, r.URL.Path)

	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
	}

	// 调用 group 的 Get 方法查找数据
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// 这里使用 protobuf 将数据编码成 protobuf 格式的二进制的 HTTP 响应
	body, err := proto.Marshal(&gcachepb.Response{Value: view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

// 客户端功能实现
// 实例化了一致性哈希算法，并且添加了传入的节点
// peers 是地址字符串数组 eg:http://127.0.0.1:9999
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// 选择远程节点客户端
// 包装了一致性哈希算法的 Get() 方法，根据具体的 key，选择节点，返回节点对应的 HTTP 客户端
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 注意这里不选择本节点
	// 因为查询缓存的逻辑是先查本地，再查远程，如果选择远程节点的时候又选了本地节点，那么会导致无限递归
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

// 远程节点客户端
// httpGetter 实现了 PeerGetter 接口
type httpGetter struct {
	baseURL string // 要访问的远程节点的地址，例如 http://example.com/_gcache/
}

// 修改 Get 方法，实现新的 protobuf 接口
func (h *httpGetter) Get(in *gcachepb.Request, out *gcachepb.Response) error {
	u := fmt.Sprintf("%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)
	// 使用 http.Get 方法获取远程节点的值
	// http 包发送 get 请求到 Cache 服务中
	// Cache 服务是实现了 ServeHTTP 方法的 HTTPPool
	// 因此被 Cache 服务的 ServeHTTP 方法捕获
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}

	// 通过 protobuf 将 res 响应的字节数据转换为 Response 结构体
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}

// 验证 httpGetter 是否实现了 PeerGetter 接口
var _ PeerGetter = (*httpGetter)(nil)
