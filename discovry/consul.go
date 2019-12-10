// consul
// 服务发现 consul 实现

package discovry

import (
	"fmt"
	consulapi "github.com/hashicorp/consul/api"
	"time"
)

type discovry struct {
	consulClient    *consulapi.Client
	resolveWaitTime time.Duration // 获取地址等待时间
}

func NewDiscovry(consulAddr string) (*discovry, error) {
	conf := consulapi.DefaultConfig()
	conf.Address = consulAddr
	c, err := consulapi.NewClient(conf)
	if err != nil {
		return nil, err
	}
	return &discovry{
		consulClient: c,
	}, nil
}

// NameResolve 获取 service 地址
func (d *discovry) NameResolve(serviceName string, tag string, resolveWaitTime time.Duration) ([]string, error) {
	q := &consulapi.QueryOptions{
		WaitTime: resolveWaitTime,
	}
	serviceEntry, _, err := d.consulClient.Health().Service(serviceName, tag, true, q)
	if err != nil {
		return nil, err
	}
	addrs := make([]string, 0)
	for i := 0; i < len(serviceEntry); i++ {
		addr := serviceEntry[i].Service.Address
		port := serviceEntry[i].Service.Port
		addrs = append(addrs, fmt.Sprintf("%s:%d", addr, port))
	}
	return addrs, nil
}

func (d *discovry) GetStrategyID(target string)(string, error){
	kp, _, err := d.consulClient.KV().Get(target,nil)
	if err != nil {
		return "", fmt.Errorf("GetStrategyID err: %v", err)
	}
	return string(kp.Value),nil
}