package balancer

import (
	"errors"
	"github.com/Sirupsen/logrus"
	dis "kelub/grpclb/discovry"
	ld "kelub/grpclb/load_reporter"
	serverpb "kelub/grpclb/pb/server"
	"sort"
	"strings"
	"time"
)

//

type Serviceer interface {
	GetServer(tags []string) (res []*ServersResponse, err error)
	GetAlladdrs(serviceName string, tags []string) ([]string, error)
	LoadClientMgr() *ld.LoadClientMgr
	// 获取负载策略
	LBStrategy() StrategyID
}

// NewService 获取新的 Service 。
// 创建服务发现以及负载管理器
func NewService(target string, serviceName string, tags []string) (Serviceer, error) {
	// TODO 配置化
	consulAddr := ":8500"
	d, err := dis.NewDiscovry(consulAddr)
	if err != nil {
		return nil, err
	}
	// TODO GetStrategy
	var service Serviceer
	strategyID := Strategy_RollPoling

	loadClientMgr := ld.NewLoadClientMgr(target)

	switch strategyID {
	case Strategy_LoadBalancer:
		service = &Service{
			target:        target,
			serviceName:   serviceName,
			tags:          tags,
			discovry:      d,
			loadClientMgr: loadClientMgr,
			strategyID:    strategyID,
		}

	case Strategy_RollPoling:
		service = &ServiceRollPoling{
			Service: Service{
				target:        target,
				serviceName:   serviceName,
				tags:          tags,
				discovry:      d,
				loadClientMgr: loadClientMgr,
				strategyID:    strategyID,
			},
			index: 0,
		}
	}
	return service, nil
}

// default Service
type Service struct {
	target      string       // serviceName+tags
	serviceName string       // serviceName 服务名
	tags        []string     //	服务 tags
	discovry    dis.Discovry //服务发现

	loadClientMgr *ld.LoadClientMgr //负载值处理

	strategyID StrategyID //负载均衡策略
}

// GetServer 获取服务器列表
func (s *Service) GetServer(tags []string) (res []*ServersResponse, err error) {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name": "Service GetServer",
		"tags":      strings.Join(tags, ","),
	})
	alladdrs, err := s.GetAlladdrs(s.serviceName, tags)
	if err != nil {
		return nil, err
	}
	logEntry.Debugf("alladdrs:[%s]", strings.Join(alladdrs, ","))
	r, err := s.loadClientMgr.GetServers(alladdrs, true)
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

func (s *Service) GetAlladdrs(serviceName string, tags []string) ([]string, error) {
	resolveWaitTime := time.Duration(1 * time.Second)
	alladdrs := make([]string, 0)
	for _, tag := range tags {
		addrs, err := s.discovry.NameResolve(serviceName, tag, resolveWaitTime)
		if err != nil {
			return nil, err
		}
		if len(addrs) == 0 || addrs == nil {
			return nil, errors.New("addrs is nil")
		}
		alladdrs = append(alladdrs, addrs...)
	}
	return alladdrs, nil
}

func (s *Service) LoadClientMgr() *ld.LoadClientMgr {
	return s.loadClientMgr
}

func (s *Service) LBStrategy() StrategyID {
	return s.strategyID
}

// Roll Poling
type ServiceRollPoling struct {
	Service
	//当前索引
	index int
}

func (s *ServiceRollPoling) rollPoling(alladdrs []string) string {
	sort.Strings(alladdrs)
	i := s.index % len(alladdrs)
	s.index++
	if s.index >= len(alladdrs) {
		s.index = 0
	}
	return alladdrs[i]
}

func (s *ServiceRollPoling) GetServer(tags []string) (res []*ServersResponse, err error) {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name": "ServiceRollPoling GetServer",
		"target":    s.target,
		"tags":      strings.Join(tags, ","),
	})
	alladdrs, err := s.GetAlladdrs(s.serviceName, tags)
	if err != nil {
		return nil, err
	}
	logEntry.Infof("index:[%d] alladdrs:[%s]", s.index, strings.Join(alladdrs, ","))
	logEntry.Infof("index:[%d]", s.index)

	addr := s.rollPoling(alladdrs)
	logEntry.Infof("addr[%s]", addr)
	sr := &ServersResponse{
		ServerAddr: addr,
		CurLoad:    0,
		State:      serverpb.ServiceStats_STARTING,
	}
	res = append(res, sr)
	return
}

func (s *ServiceRollPoling) LoadClientMgr() *ld.LoadClientMgr {
	return s.loadClientMgr
}

func (s *ServiceRollPoling) LBStrategy() StrategyID {
	return s.strategyID
}
