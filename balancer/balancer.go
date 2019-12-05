// Balancer
// The main func
// Get server

package balancer

import (
	"github.com/Sirupsen/logrus"
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
	GetServers(serviceName string, tags []string, hashID uint64) ([]*ServersResponse, error)
}

type Balancer struct {
	Serverslist *sync.Map //target: Serviceer
	Config *Config
}

func (b *Balancer) RefreshAllLoad() {
	b.Serverslist.Range(func(key, value interface{}) bool {
		target := key.(string)
		service := value.(Serviceer)
		if err := b.refreshLoad(target, service);err != nil{
			return false
		}
		return true
	})
}

func (b *Balancer) refresloop() {
	t := time.NewTicker(b.Config.Balancer.refreshInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			b.RefreshAllLoad()
		}
	}
}

// refreshLoad 刷新负载值
func (b *Balancer) refreshLoad(target string, service Serviceer) error {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name": "updateLoad",
		"target":    target,
	})
	serviceName, tags := b.targetToName(target)
	alladdrs, err := service.GetAlladdrs(serviceName, tags)
	if err != nil {
		b.deleteServiceCache(target)
		logEntry.Errorln("[GetAlladdrs] service target:[target] err: ", err)
		return err
	}
	if service.LBStrategy() == Strategy_LoadBalancer {
		_, err = service.LoadClientMgr().GetServers(alladdrs, false)
		if err != nil {
			logEntry.Errorln("GetServers err:", err)
			return err
		}
		sort.Strings(alladdrs)
		removeaddrs := make([]string, 0)
		service.LoadClientMgr().LoadClientList.Range(func(key, value interface{}) bool {
			oldaddr := key.(string)
			index := sort.SearchStrings(alladdrs, oldaddr)
			//SearchStrings在递增顺序的alladdrs中搜索oldaddr，返回oldaddr的索引。
			//如果查找不到，返回值是oldadd应该插入alladdrs的位置
			//返回值可以是len(alladdrs)。
			if index >= len(alladdrs) || alladdrs[index] != oldaddr {
				removeaddrs = append(removeaddrs, oldaddr)
				service.LoadClientMgr().DeleteCache(oldaddr)
			}
			return true
		})
	}
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
	config := DefaultConfig()
	b := &Balancer{
		Serverslist: new(sync.Map),
		Config: config,
	}
	go b.refresloop()
	return b
}

// GetServers获取服务器信息列表
func (b *Balancer) GetServers(serviceName string, tags []string, hashID uint64) ([]*ServersResponse, error) {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name":   "GetServers",
		"serviceName": serviceName,
	})
	target := b.nameToTarget(serviceName, tags)
	logEntry.Debugln("target:", target)
	s, ok := b.Serverslist.Load(target)
	if ok {
		service := s.(Serviceer)
		server, err := service.GetServer(tags)
		if err != nil {
			b.Serverslist.Delete(target)
			return nil, err
		}
		return server, nil
	}

	service, err := NewService(target, serviceName, tags, hashID,b.Config.Discovry.consulAddr)
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
