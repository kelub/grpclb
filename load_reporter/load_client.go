package load_reporter

import (
	"context"
	serverpb "kelub/grpclb/pb/server"
	"sync"
	"sync/atomic"
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

type LoadResult struct {
	Error    error
	Addr     string
	Response *serverpb.LoadReporterResponse
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

func (lcm *LoadClientMgr) getServer(ctx context.Context, lc *LoadClient) chan *LoadResult {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name":   "getServer",
		"serviceAddr": lc.serviceAddr,
	})
	rch := make(chan *LoadResult)

	defer close(rch)
	lrr, err := lc.GetLoad(ctx)
	if err != nil {
		logEntry.Error(err)
		rch <- &LoadResult{
			Error:    err,
			Addr:     lc.serviceAddr,
			Response: nil,
		}
		return rch
	}
	lcm.LoadReporterResList.Store(lc.serviceAddr, &loadResTime{
		lrr, time.Now(),
	})
	rch <- &LoadResult{
		Error:    nil,
		Addr:     lc.serviceAddr,
		Response: lrr,
	}
	return rch
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
	// rch := make(chan map[string]*serverpb.LoadReporterResponse, len(serviceAddrs))

	LoadResult := make([]<-chan *LoadResult, len(serviceAddrs))
	ctx, cancel := context.WithTimeout(context.Background(), getServerTimeout)
	defer cancel()
	var count int64 = 0
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
			// go lcm.getServer1(ctx, lc, addr)
			go func() {
				LoadResult[atomic.LoadInt64(&count)] = lcm.getServer(ctx, lc)
				atomic.AddInt64(&count, 1)
			}()
		}
		select {
		case <-ctx.Done():
			logEntry.Error("getServer timeout advance")
			return r, nil
		default:
		}
	}
	if count > 0 {
		for i := range LoadResult {
			loadResultCh := LoadResult[i]
			select {
			case <-ctx.Done():
				logEntry.Error("getServer timeout")
				return r, nil
			case loadResult := <-loadResultCh:
				if loadResult.Error == nil {
					r[loadResult.Addr] = loadResult.Response
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
