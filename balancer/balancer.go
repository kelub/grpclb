// Balancer
// The main func
// Get server

package balancer

import (
	"kelub/grpclb/discovry"
	ld "kelub/grpclb/load_reporter"
	serverpb "kelub/grpclb/pb/server"
	"strings"
	"sync"
)

type ServersResponse struct {
	ServerAddr string
	CurLoad    int64
	State      serverpb.ServiceStats
}

type Balancerer interface {
	GetServers(serviceName string, tags []string) ([]*ServersResponse, error)
}

type Balancer struct {
	Serverslist *sync.Map
}

func (b *Balancer) nameToTarget(serviceName string, tags []string) (res string) {
	if tags == nil {
		return serviceName
	}
	return serviceName + "," + strings.Join(tags, ",")
}

func (b *Balancer) targetToName(target string) (serviceName string, tags []string) {
	s := strings.Split(target, ",")
	if len(s) == 1 {
		return s[0], nil
	}
	serviceName = s[0]
	tags = s[1:]
	return
}

func NewBalancer() *Balancer {
	return &Balancer{
		Serverslist: new(sync.Map),
	}
}

func (b *Balancer) GetServers(serviceName string, tags []string) ([]*ServersResponse, error) {
	target := b.nameToTarget(serviceName, tags)
	s, ok := b.Serverslist.Load(target)
	if ok {
		server := s.([]*ServersResponse)
		return server, nil
	}

}

type Service struct {
	discovry discovry.Discovry
	load     *ld.LoadBlancerReporter
}

func NewService() (*Service, error) {
	consulAddr := ""
	discovry, err := discovry.NewDiscovry(consulAddr)
	if err != nil {
		return nil, err
	}

	//loadClient

	return &Service{}, nil
}
