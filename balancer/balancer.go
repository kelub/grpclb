// Balancer
// The main func
// Get server

package balancer

import (
	dis "kelub/grpclb/discovry"
	ld "kelub/grpclb/load_reporter"
	serverpb "kelub/grpclb/pb/server"
	"strings"
	"sync"
	"time"
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
	//target := b.nameToTarget(serviceName, tags)
	//s, ok := b.Serverslist.Load(target)
	//if ok {
	//	server := s.([]*ServersResponse)
	//	return server, nil
	//}
	////servers, err :=

}

type Service struct {
	target   string
	address  []string
	serviceName string
	tags 	[]string
	discovry dis.Discovry

	loadClientMgr *ld.LoadClientMgr
}

func NewService(target string, serviceName string, tags []string) (*Service, error) {
	consulAddr := ""
	d, err := dis.NewDiscovry(consulAddr)
	if err != nil {
		return nil, err
	}

	//loadClient
	loadClientMgr := ld.NewLoadClientMgr(target)
	return &Service{
		serviceName: serviceName,
		tags: tags,
		discovry:      d,
		loadClientMgr: loadClientMgr,
	}, nil
}

func (s *Service) GetServer(tags []string) {
	resolveWaitTime := time.Duration(3 * time.Second)
	alladdrs := make([]string, 0)
	for _, tag := range tags {
		addrs, err := s.discovry.NameResolve(s.serviceName,tag,resolveWaitTime)
		if err != nil {
			return nil, err
		}
		alladdrs := append(alladdrs, addrs...)
	}
}
