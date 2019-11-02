package load_reporter

import (
	"context"
	serverpb "kelub/grpclb/pb/server"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	LoadCacheInterval = 30 * time.Second
)

type LoadClientMgrer interface {
}

// LoadClient 负载
type LoadClient struct {
	cc  *grpc.ClientConn
	lrc serverpb.LoadReporterServiceClient
	//ctx context.Context
	serviceAddr string
	//load        int64
	//state       serverpb.ServiceStats

	createdAt time.Time
}

func NewLoadClient(ctx context.Context, serviceAddr string) (*LoadClient, error) {
	conn, err := grpc.Dial(serviceAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	lrc := serverpb.NewLoadReporterServiceClient(conn)
	return &LoadClient{
		cc:          conn,
		lrc:         lrc,
		serviceAddr: serviceAddr,
		//ctx:         ctx,

		createdAt: time.Now(),
	}, nil
}

// GetLoad 获取负载
func (lc *LoadClient) GetLoad(ctx context.Context) (r *serverpb.LoadReporterResponse, err error) {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name": "GetLoad",
	})
	//ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	//defer cancel()
	r, err = lc.lrc.LoadReporter(ctx, &serverpb.LoadReporterRequest{})
	if err != nil {
		logEntry.Errorf("could not verify: %v", err)
		return nil, err
	}
	logEntry.Infof("addr:[%s] curload[%d]", lc.serviceAddr, r.CurLoad)
	return
}

//func (lc *LoadClient) GetCurLoad() (int64, error) {
//	lrr, err := lc.GetLoad()
//	if err != nil {
//		return 0, err
//	}
//	return lrr.GetCurLoad(), nil
//}

// LoadClientMgr 管理一组 service 负载
type LoadClientMgr struct {
	//One group target
	target              string
	serviceAddrs        []string
	LoadClientList      *sync.Map //addr: *LoadClient
	LoadReporterResList *sync.Map //addr: *serverpb.LoadReporterResponse ,time.now()

	loadCacheInterval time.Duration
}

type loaderror struct {
	serviceAddr string
	err         error
}

type loadResTime struct {
	loadRes   *serverpb.LoadReporterResponse
	createdAt time.Time
}

func NewLoadClientMgr(target string) *LoadClientMgr {
	lcm := &LoadClientMgr{
		target:              target,
		LoadClientList:      new(sync.Map),
		LoadReporterResList: new(sync.Map),

		loadCacheInterval: LoadCacheInterval,
	}
	return lcm
}

func (lcm *LoadClientMgr) getServer(ctx context.Context,
	errch chan *loaderror, rch chan map[string]*serverpb.LoadReporterResponse,
	lc *LoadClient, serviceAddr string) error {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name":   "getServer",
		"serviceAddr": serviceAddr,
	})
	addrLrr := make(map[string]*serverpb.LoadReporterResponse)
	lrr, err := lc.GetLoad(ctx)
	if err != nil {
		logEntry.Error(err)
		e := &loaderror{
			serviceAddr: serviceAddr,
			err:         err,
		}
		errch <- e
		return err
	}
	lcm.LoadReporterResList.Store(serviceAddr, &loadResTime{
		lrr, time.Now(),
	})
	addrLrr[serviceAddr] = lrr
	rch <- addrLrr

	return nil
}

// GetServers 获取服务负载列表
// LoadReporterResList cache
// LoadClientList cache -> getServer
func (lcm *LoadClientMgr) GetServers(serviceAddrs []string, useResCache bool) (r map[string]*serverpb.LoadReporterResponse, err error) {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name":    "GetServers",
		"serviceAddrs": serviceAddrs,
	})
	// TODO 配置化
	getServerTimeout := 100 * time.Millisecond
	lcm.serviceAddrs = serviceAddrs
	r = make(map[string]*serverpb.LoadReporterResponse)
	rch := make(chan map[string]*serverpb.LoadReporterResponse, len(serviceAddrs))
	errch := make(chan *loaderror)
	ctx, cancel := context.WithTimeout(context.Background(), getServerTimeout)
	defer cancel()
	var count = 0
	var rcount = 0
	now := time.Now()
	for i := range serviceAddrs {
		var isNewLoadClient = true
		addr := serviceAddrs[i]
		l, ok := lcm.LoadReporterResList.Load(addr)
		if ok {
			if l.(*loadResTime).createdAt.Sub(now) > lcm.loadCacheInterval {
				lcm.LoadReporterResList.Delete(addr)
				isNewLoadClient = true
			} else if useResCache {
				r[addr] = l.(*loadResTime).loadRes
				logEntry.Info("cache!")
				isNewLoadClient = false
			}
		}

		if isNewLoadClient {
			var lc *LoadClient
			v, ok := lcm.LoadClientList.Load(addr)
			if ok {
				lc = v.(*LoadClient)
			} else {
				lc, err = NewLoadClient(ctx, addr)
				if err != nil {
					return nil, err
				}
				lcm.LoadClientList.Store(addr, lc)
			}
			go lcm.getServer(ctx, errch, rch, lc, addr)
			count++
		}

	}
	if count > 0 {
		for {
			select {
			case e := <-errch:
				logEntry.Errorln(e.err)
				lcm.DeleteCache(e.serviceAddr)
				return nil, e.err
			case <-ctx.Done():
				logEntry.Error("getServer timeout")
				return nil, nil
			case rll := <-rch:
				for k, v := range rll {
					r[k] = v
				}
				rcount++
				if rcount == count {
					return
				}
			}
		}

	}
	return
}

func (lcm *LoadClientMgr) DeleteCache(serviceAddr string) {
	//delete lcm.LoadClientList
	//delete lcm.LoadReporterResList
	lcm.LoadReporterResList.Delete(serviceAddr)
	lcm.LoadClientList.Delete(serviceAddr)
}
