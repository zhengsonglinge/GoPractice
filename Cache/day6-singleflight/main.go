package main

import (
	"flag"
	"fmt"
	"gcache"
	"log"
	"net/http"
)

// map 模拟数据源
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

/*
gcache/
    |--lru/
        |--lru.go  // lru 缓存淘汰策略
    |--byteview.go // 缓存值的抽象与封装
    |--cache.go    // 并发控制
    |--gcache.go   // 负责与外部交互，控制缓存存储和获取的主流程
	|--http.go     // 提供被其他节点访问的能力(基于http)
*/
/*
                            是
接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
                |  否                         是
                |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
                            |  否
                            |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
*/

// 创建一个名为 scores 的 Group 缓存空间，如果缓存未命中则读取数据库并返回
func createGroup() *gcache.Group {
	return gcache.NewGroup("scores", 2<<10, gcache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// 启动缓存服务器，创建 HTTPPool，添加节点信息，注册到 httpPool 中，启动 HTTP 服务（共3个端口，8001/8002/8003），用户不感知。
// 三个端口用来代表三个远程节点
func startCacheServer(addr string, addrs []string, group *gcache.Group) {
	// peers 是 HTTPPool，实现了 PeerPicker 接口和 http.Handler 接口
	peers := gcache.NewHTTPPool(addr)
	peers.Set(addrs...)
	// 注册 peers 用来选择远程节点
	group.RegisterPeers(peers)
	log.Println("gcache is running at", addr)
	// peers 用来代理发送到当前节点的 http 请求，同样也是调用 group 的 Get 请求获取本地和远程数据
	// API 服务调用 group 的 Get 方法，先查本地，再查远程。Cache 服务也调用 group 的 Get 方法
	// 这就是为什么一致性哈希在选择节点的时候不选择本地节点，防止无限递归
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

// 启动一个 API 服务（端口 9999），与用户进行交互，用户感知。
// API 服务通过 gache.Group 的 Get 方法获取本地和远程节点的缓存
func startAPISever(apiAddr string, group *gcache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			// 先查本地缓存，本地没有再选择远程节点，调用远程节点 Get 方法获取数据
			view, err := group.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(append(view.ByteSlice(), '\n'))
		},
	))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

// 两个服务，API 服务、Cache 服务
// API 服务调用 group 的 Get 方法，先查本地，再查远程。
// API 查远程的时候调用的是实现了 PeerPicker 接口的 HTTPPool
// Cache 服务通过 HTTPPool 接口的 ServeHTTP 函数接收 API 请求，也调用 group 的 Get 方法查询数据
// Cache 服务也是先查询本地，再查询远程 Cache 服务
// 因为是同一个 key，一致性哈希选择的也是同一个远程节点，因此 Cache 选择远程节点时选择的就是本地节点了
// 再次选择本地节点说明缓存未命中，调用回调函数从外部获取数据
// 不论是 API 还是 Cache 服务，在调用 group 的 Get 方法、选择远程节点的时候都不会选择本节点了，因为选择本节点可能导致无限递归
func main() {

	var port int
	var api bool

	// 用于命令行获取参数
	flag.IntVar(&port, "port", 8001, "gcache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	// 启动 api 服务
	apiAddr := "http://localhost:9999"
	group := createGroup()
	if api {
		// 命令行中只一条命令是 api=true 的，因此只启动一个 9999 端口的 API 服务
		// 这里的 group 用来查询缓存
		go startAPISever(apiAddr, group)
	}

	// 启动 cache 服务
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	// 每次启动一个端口作为一个 Cache 节点，每个 Cache 节点都注册三个远程节点（包括自己）
	// 命令行启动三次，即启动三个 Cache 节点
	// 这里的 group 用来注册远程节点
	startCacheServer(addrMap[port], addrs, group)
}

/*
./run.sh
2024/04/10 17:58:13 gcache is running at http://localhost:8003
2024/04/10 17:58:13 gcache is running at http://localhost:8002
2024/04/10 17:58:13 fontend server is running at http://localhost:9999
2024/04/10 17:58:13 gcache is running at http://localhost:8001
>>> start test
2024/04/10 17:58:15 [Server http://localhost:8003] Pick peer http://localhost:8001
2024/04/10 17:58:15 [Server http://localhost:8001] GET /_gcache/scores/Tom
2024/04/10 17:58:15 [SlowDB] search key Tom
630
2024/04/10 17:58:15 [Server http://localhost:8003] Pick peer http://localhost:8001
2024/04/10 17:58:15 [Server http://localhost:8001] GET /_gcache/scores/Tom
2024/04/10 17:58:15 [GCache] hit
630
2024/04/10 17:58:15 [Server http://localhost:8003] Pick peer http://localhost:8001
2024/04/10 17:58:15 [Server http://localhost:8001] GET /_gcache/scores/Tom
2024/04/10 17:58:15 [GCache] hit
630
*/
