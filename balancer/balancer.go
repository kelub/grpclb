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

func (b *Balancer) init() {
	b.Serverslist.Range(func(key, value interface{}) bool {
		addr := key.(string)
		service := value.(*Service)

	})
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
		service := s.(*Service)
		server, err := service.GetServer(tags)
		if err != nil {
			b.Serverslist.Delete(target)
			return nil, err
		}
		return server, nil
	}

	service, err := NewService(target, serviceName, tags)
	if err != nil {
		return nil, err
	}
	b.Serverslist.Store(target, service)
	server, err := service.GetServer(tags)
	if err != nil {
		b.Serverslist.Delete(target)
		return nil, err
	}

	//TODO load value handler
	return server, nil
}

type Service struct {
	target      string
	address     []string
	serviceName string
	tags        []string
	discovry    dis.Discovry

	loadClientMgr *ld.LoadClientMgr
}

func NewService(target string, serviceName string, tags []string) (*Service, error) {
	consulAddr := ":8500"
	d, err := dis.NewDiscovry(consulAddr)
	if err != nil {
		return nil, err
	}

	//loadClient
	loadClientMgr := ld.NewLoadClientMgr(target)
	return &Service{
		target:        target,
		serviceName:   serviceName,
		tags:          tags,
		discovry:      d,
		loadClientMgr: loadClientMgr,
	}, nil
}

func (s *Service) GetServer(tags []string) (res []*ServersResponse, err error) {
	resolveWaitTime := time.Duration(1 * time.Second)
	alladdrs := make([]string, 0)
	for _, tag := range tags {
		addrs, err := s.discovry.NameResolve(s.serviceName, tag, resolveWaitTime)
		if err != nil {
			return nil, err
		}
		alladdrs = append(alladdrs, addrs...)
	}
	r, err := s.loadClientMgr.GetServers(alladdrs)
	if err != nil {
		return nil, err
	}
	for k, v := range r {
		sr := &ServersResponse{
			ServerAddr: k,
			CurLoad:    v.CurLoad,
			State:      v.State,
		}
		res = append(res, sr)
	}
	return res, err
}
