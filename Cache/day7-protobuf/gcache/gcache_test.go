package gcache

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

func TestGetter(t *testing.T) {
	// 将匿名函数类型转换成 Getter
	// 调用 f 即调用匿名函数
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})
	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback failed")
	}
}

// 接口型函数的价值在于:
// 既能够将普通的函数类型（需类型转换）作为参数，也可以将结构体作为参数
// 使用更为灵活，可读性也更好
func TestImplementFunc(t *testing.T) {
	// 使用接口型函数，匿名函数类型换转为 GetterFunc，GetterFunc 是实现了 Getter 接口的接口型函数
	// 既可以使用匿名函数，也可以使用具名函数，只要参数和返回值与 Getter 接口的 Get 函数相同即可
	// 接口型函数要求接口只有一个方法
	// 接口型函数在调用实现的接口方法 Get 时调用的就是接口型函数自身
	GetFromSource(GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	}), "key")

	// 也可以传入实现了 Getter 接口的结构体
	// 适用于逻辑复杂的请求，比如数据库访问等需要很多参数的情况
	GetFromSource(new(DB), "key")
}

func GetFromSource(getter Getter, key string) []byte {
	buf, err := getter.Get(key)
	if err != nil {
		return nil
	}
	return buf
}

type DB struct {
	url string
}

func (db *DB) Get(key string) ([]byte, error) {
	return []byte(key + db.url), nil
}

// 创建示例测试 Group
func TestGroup(t *testing.T) {

	// map 模拟数据库
	var db = map[string]string{
		"Tom":  "630",
		"Jack": "589",
		"Sam":  "567",
	}

	// 用来记录加载次数的
	loadCounts := make(map[string]int, len(db))

	gc := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB search key]", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key] += 1
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		},
	))

	for k, v := range db {
		// 第一次缓存未命中，执行回调函数，将 db 中数据加载到内存
		if view, err := gc.Get(k); err != nil || view.String() != v {
			t.Fatalf("failed to get value of Tom")
		}
		// 第二次缓存命中且加载次数不超过 1
		if _, err := gc.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		}
	}

	if view, err := gc.Get("unkown"); err == nil {
		t.Fatalf("the value of unkown should be empty,but %s got", view)
	}
}

/*
=== RUN   TestGroup
2024/04/10 00:20:50 [SlowDB search key] Tom
2024/04/10 00:20:50 [GCache] hit
2024/04/10 00:20:50 [SlowDB search key] Jack
2024/04/10 00:20:50 [GCache] hit
2024/04/10 00:20:50 [SlowDB search key] Sam
2024/04/10 00:20:50 [GCache] hit
2024/04/10 00:20:50 [SlowDB search key] unkown
--- PASS: TestGroup (0.04s)
PASS
*/
