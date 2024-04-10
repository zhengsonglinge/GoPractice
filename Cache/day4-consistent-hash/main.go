package main

import (
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
func main() {
	// 创建一个名为 scores 的 Group 缓存空间，如果缓存未命中则读取数据库并返回
	gcache.NewGroup("scores", 2<<10, gcache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	// peers 代理 http 请求
	addr := "localhost:9999"
	peers := gcache.NewHTTPPool(addr)
	log.Println("gcache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}

/*
客户端
curl http://localhost:9999/_gcache/scores/Tom
630
curl http://localhost:9999/_gcache/scores/kkk
kkk not exist

服务端
go run main.go
2024/04/10 01:33:21 gcache is running at localhost:9999
2024/04/10 01:33:24 [Server localhost:9999] GET /_gcache/scores/Tom
2024/04/10 01:33:24 [SlowDB] search key Tom
2024/04/10 01:33:45 [Server localhost:9999] GET /_gcache/scores/Tom
2024/04/10 01:33:45 [GCache] hit
2024/04/10 01:34:07 [Server localhost:9999] GET /_gcache/scores/kkk
2024/04/10 01:34:07 [SlowDB] search key kkk
*/
