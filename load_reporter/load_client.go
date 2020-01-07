package load_reporter

import (
	"context"
	serverpb "kelub/grpclb/pb/server"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
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
	LoadReporterResList serversResult

	refreshInterval  time.Duration
	getServerTimeout time.Duration

	cancel context.CancelFunc
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

type serversResult map[string]*serverpb.LoadReporterResponse

func NewLoadClientMgr(target string, refreshInterval, getServerTimeout time.Duration) *LoadClientMgr {
	ctx, cancel := context.WithCancel(context.Background())
	lcm := &LoadClientMgr{
		target:         target,
		LoadClientList: new(sync.Map),

		refreshInterval:  refreshInterval,
		getServerTimeout: getServerTimeout,
		cancel:           cancel,
	}
	go lcm.refresloop(ctx)
	return lcm
}

func (lcm *LoadClientMgr) getServer(ctx context.Context, lc *LoadClient) chan *LoadResult {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name":   "getServer",
		"serviceAddr": lc.serviceAddr,
	})
	rch := make(chan *LoadResult)
	defer close(rch)
	go func() {
		select {
		case <-ctx.Done():
			rch <- &LoadResult{
				Error:    ctx.Err(),
				Addr:     lc.serviceAddr,
				Response: nil,
			}
			return
		default:
		}

		lrr, err := lc.GetLoad(ctx)
		if err != nil {
			logEntry.Error(err)
			rch <- &LoadResult{
				Error:    err,
				Addr:     lc.serviceAddr,
				Response: nil,
			}
			return
		}
		//lcm.LoadReporterResList.Store(lc.serviceAddr, &loadResTime{
		//	lrr, time.Now(),
		//})
		rch <- &LoadResult{
			Error:    nil,
			Addr:     lc.serviceAddr,
			Response: lrr,
		}
	}()
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
	logEntry.Info("")
	lcm.serviceAddrs = serviceAddrs
	return lcm.LoadReporterResList, nil
}

func (lcm *LoadClientMgr) DeleteCache(serviceAddr string) {
	//delete lcm.LoadClientList
	lcm.LoadClientList.Delete(serviceAddr)
}

func (lcm *LoadClientMgr) refresloop(ctx context.Context) {
	t := time.NewTicker(lcm.refreshInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			lcm.LoadReporterResList, _ = lcm.getServers(ctx, lcm.serviceAddrs)
		}
	}
}

func (lcm *LoadClientMgr) getServers(ctx context.Context, serviceAddrs []string) (r serversResult, err error) {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name":    "getServers",
		"serviceAddrs": serviceAddrs,
	})
	if serviceAddrs == nil || len(serviceAddrs) == 0 {
		return nil, nil
	}
	r = make(map[string]*serverpb.LoadReporterResponse)
	LoadResultChs := make([]<-chan *LoadResult, len(serviceAddrs))
	ctx, cancel := context.WithTimeout(ctx, lcm.getServerTimeout)
	defer cancel()
	for i := range serviceAddrs {
		addr := serviceAddrs[i]
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

		LoadResultChs[i] = lcm.getServer(ctx, lc)

		select {
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				logEntry.Info("getServers canceled")
			} else if ctx.Err() == context.DeadlineExceeded {
				logEntry.Error("getServer timeout advance", ctx.Err())
			}
			return r, ctx.Err()
		default:
		}
	}
	// 获取所有结果，直到取完。或者 ctx.Done()返回 超时退出等情况。
	for lr := range lcm.funIn(ctx, LoadResultChs...) {
		r[lr.Addr] = lr.Response
	}

	return
}

// 扇入模式
func (lcm *LoadClientMgr) funIn(ctx context.Context, in ...<-chan *LoadResult) <-chan *LoadResult {
	wg := sync.WaitGroup{}
	out := make(chan *LoadResult)
	multiplex := func(c <-chan *LoadResult) {
		defer wg.Done()
		for i := range c {
			select {
			case <-ctx.Done():
				return
			case out <- i:
			}
		}
	}

	// 读取值
	for _, c := range in {
		wg.Add(1)
		go multiplex(c)
	}

	// 当所有值获取完后 关闭 channle
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func (lcm *LoadClientMgr) Stop() {
	lcm.cancel()
}
