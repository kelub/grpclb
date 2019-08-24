// Balancer
// The main func
// Get server

package balancer

import (
	"github.com/Sirupsen/logrus"
	dis "kelub/grpclb/discovry"
	ld "kelub/grpclb/load_reporter"
	serverpb "kelub/grpclb/pb/server"
	"sort"
	"strings"
	"sync"
	"time"
)

// 负载返回结构体
type ServersResponse struct {
	ServerAddr string                // 服务器地址
	CurLoad    int64                 // 服务器当前负载值
	State      serverpb.ServiceStats // 服务器当前状态
}

type Balancerer interface {
	GetServers(serviceName string, tags []string) ([]*ServersResponse, error)
}

type Balancer struct {
	Serverslist *sync.Map //target: *Service
}

func (b *Balancer) RefreshAllLoad() {
	b.Serverslist.Range(func(key, value interface{}) bool {
		target := key.(string)
		service := value.(*Service)
		b.refreshLoad(target, service)
		return true
	})
}

func (b *Balancer) refresloop() {
	// TODO 配置化 refreshInterval
	refreshInterval := 5 * time.Second
	t := time.NewTicker(refreshInterval)
	for {
		select {
		case <-t.C:
			b.RefreshAllLoad()
		}
	}
}

// refreshLoad 刷新负载值
func (b *Balancer) refreshLoad(target string, service *Service) error {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name": "updateLoad",
		"target":    target,
	})
	serviceName, tags := b.targetToName(target)
	alladdrs, err := service.getAlladdrs(serviceName, tags)
	if err != nil {
		b.deleteServiceCache(target)
		logEntry.Errorf("[getAlladdrs] service target:[target] err: ", err)
		return err
	}
	_, err = service.loadClientMgr.GetServers(alladdrs, false)
	if err != nil {
		logEntry.Errorf("GetServers err:", err)
		return err
	}
	sort.Strings(alladdrs)
	removeaddrs := make([]string, 0)
	service.loadClientMgr.LoadClientList.Range(func(key, value interface{}) bool {
		oldaddr := key.(string)
		index := sort.SearchStrings(alladdrs, oldaddr)
		//SearchStrings在递增顺序的alladdrs中搜索oldaddr，返回oldaddr的索引。
		//如果查找不到，返回值是oldadd应该插入alladdrs的位置
		//返回值可以是len(alladdrs)。
		if index >= len(alladdrs) || alladdrs[index] != oldaddr {
			removeaddrs = append(removeaddrs, oldaddr)
			service.loadClientMgr.DeleteCache(oldaddr)
		}
		return true
	})
	return err
}

func (b *Balancer) deleteServiceCache(target string) {
	b.Serverslist.Delete(target)
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

// NewBalancer 创建Balancer，并开启定时刷新循环，返回Balancer。
func NewBalancer() *Balancer {
	b := &Balancer{
		Serverslist: new(sync.Map),
	}
	go b.refresloop()
	return b
}

// GetServers获取服务器信息列表
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

// Service
type Service struct {
	target      string       // serviceName+tags
	serviceName string       // serviceName 服务名
	tags        []string     //	服务 tags
	discovry    dis.Discovry //服务发现

	loadClientMgr *ld.LoadClientMgr //负载值处理
}

// NewService 获取新的 Service 。
// 创建服务发现以及负载管理器
func NewService(target string, serviceName string, tags []string) (*Service, error) {
	// TODO 配置化
	consulAddr := ":8500"
	d, err := dis.NewDiscovry(consulAddr)
	if err != nil {
		return nil, err
	}
	loadClientMgr := ld.NewLoadClientMgr(target)
	return &Service{
		target:        target,
		serviceName:   serviceName,
		tags:          tags,
		discovry:      d,
		loadClientMgr: loadClientMgr,
	}, nil
}

// GetServer 获取服务器列表
func (s *Service) GetServer(tags []string) (res []*ServersResponse, err error) {
	alladdrs, err := s.getAlladdrs(s.serviceName, tags)
	if err != nil {
		return nil, err
	}
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

func (s *Service) getAlladdrs(serviceName string, tags []string) ([]string, error) {
	resolveWaitTime := time.Duration(1 * time.Second)
	alladdrs := make([]string, 0)
	for _, tag := range tags {
		addrs, err := s.discovry.NameResolve(serviceName, tag, resolveWaitTime)
		if err != nil {
			return nil, err
		}
		alladdrs = append(alladdrs, addrs...)
	}
	return alladdrs, nil
}
