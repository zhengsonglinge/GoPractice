package xclient

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type GrpcRegistryDiscovery struct {
	*MultiServersDiscovery               // GrpcRegistryDiscovery 嵌套了 MultiServersDiscovery，很多能力可以复用。
	registry               string        // registry 即注册中心的地址
	timeout                time.Duration // timeout 服务列表的过期时间
	lastUpdate             time.Time     // lastUpdate 是代表最后从注册中心更新服务列表的时间，默认 10s 过期，即 10s 之后，需要从注册中心更新新的列表。
}

const defaultUpdateTimeout = time.Second * 10

func NewGrpcRegistryDiscovery(registerAddr string, timeout time.Duration) *GrpcRegistryDiscovery {
	if timeout == 0 {
		timeout = defaultUpdateTimeout
	}
	d := &GrpcRegistryDiscovery{
		MultiServersDiscovery: NewMultiServerDiscovery(make([]string, 0)),
		registry:              registerAddr,
		timeout:               timeout,
	}
	return d
}

// 实现 Update 和 Refresh 方法，
// 超时重新获取的逻辑在 Refresh 中实现
func (d *GrpcRegistryDiscovery) Update(servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = servers
	d.lastUpdate = time.Now()
	return nil
}

func (d *GrpcRegistryDiscovery) Refresh() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.lastUpdate.Add(d.timeout).After(time.Now()) {
		return nil
	}
	log.Println("rpc registry: refresh servers from registry", d.registry)
	resp, err := http.Get(d.registry)
	if err != nil {
		log.Println("rpc registry refresh err:", err)
		return err
	}
	servers := strings.Split(resp.Header.Get("X-Geerpc-Servers"), ",")
	d.servers = make([]string, 0, len(servers))
	for _, server := range servers {
		if strings.TrimSpace(server) != "" {
			d.servers = append(d.servers, strings.TrimSpace(server))
		}
	}
	d.lastUpdate = time.Now()
	return nil
}

// Get 和 GetAll 与 MultiServersDiscovery 相似，
// 唯一的不同在于，GeeRegistryDiscovery 需要先调用 Refresh 确保服务列表没有过期
func (d *GrpcRegistryDiscovery) Get(mode SelectMode) (string, error) {
	if err := d.Refresh(); err != nil {
		return "", err
	}
	return d.MultiServersDiscovery.Get(mode)
}

func (d *GrpcRegistryDiscovery) GetAll() ([]string, error) {
	if err := d.Refresh(); err != nil {
		return nil, err
	}
	return d.MultiServersDiscovery.GetAll()
}
