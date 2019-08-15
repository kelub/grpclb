// consul

package discovry

import (
	"fmt"
	consulapi "github.com/hashicorp/consul/api"
)

type discovry struct {
	consulClient *consulapi.Client
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

func (d *discovry) NameResolve(serviceName string, tags string) ([]string, error) {
	serviceEntry, _, err := d.consulClient.Health().Service(serviceName, tags, true, nil)
	if err != nil {
		return nil, err
	}
	addrs := make([]string, len(serviceEntry))
	for i := 0; i <= len(serviceEntry); i++ {
		addr := serviceEntry[i].Service.Address
		port := serviceEntry[i].Service.Port
		addrs[i] = fmt.Sprintf("%s:%d", addr, port)
	}
	return addrs, nil
}
