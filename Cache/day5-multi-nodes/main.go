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
// 启动三个端口用来代表三个远程节点
func startCacheServer(addr string, addrs []string, gc *gcache.Group) {
	peers := gcache.NewHTTPPool(addr)
	peers.Set(addrs...)
	gc.RegisterPeers(peers)
	log.Println("gcache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

// 启动一个 API 服务（端口 9999），与用户进行交互，用户感知。
func startAPISever(apiAddr string, gc *gcache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := gc.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		},
	))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func main() {

	var port int
	var api bool

	// 用于命令行获取参数
	flag.IntVar(&port, "port", 8001, "gcache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	// 启动 api 服务
	apiAddr := "http://localhost:9999"
	g := createGroup()
	if api {
		go startAPISever(apiAddr, g)
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

	startCacheServer(addrMap[port], []string(addrs), g)
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
