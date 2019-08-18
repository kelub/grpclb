// consul

package discovry

import (
	"fmt"
	consulapi "github.com/hashicorp/consul/api"
	"time"
)

type discovry struct {
	consulClient    *consulapi.Client
	resolveWaitTime time.Duration
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
