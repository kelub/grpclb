package load_reporter

import (
	"context"
	"fmt"
	"github.com/Sirupsen/logrus"
	"google.golang.org/grpc"
	serverpb "kelub/grpclb/pb/server"
	"sync"
	"time"
)

type LoadClientMgrer interface {
}

type LoadClient struct {
	cc  *grpc.ClientConn
	lrc serverpb.LoadReporterServiceClient
	//ctx context.Context
	serviceAddr string
	//load        int64
	//state       serverpb.ServiceStats
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
	}, nil
}

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
	logEntry.Infof("[%d]", r.CurLoad)
	return
}

//func (lc *LoadClient) GetCurLoad() (int64, error) {
//	lrr, err := lc.GetLoad()
//	if err != nil {
//		return 0, err
//	}
//	return lrr.GetCurLoad(), nil
//}

type LoadClientMgr struct {
	//One group target
	target              string
	serviceAddrs        []string
	LoadClientList      *sync.Map //addr: *LoadClient
	LoadReporterResList *sync.Map //addr: *serverpb.LoadReporterResponse
}

type loaderror struct {
	serviceAddr string
	err         error
}

func NewLoadClientMgr(target string) *LoadClientMgr {
	lcm := &LoadClientMgr{
		target:              target,
		LoadClientList:      new(sync.Map),
		LoadReporterResList: new(sync.Map),
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
	lcm.LoadReporterResList.Store(serviceAddr, lrr)
	addrLrr[serviceAddr] = lrr
	rch <- addrLrr

	return nil
}

func (lcm *LoadClientMgr) GetServers(serviceAddrs []string, useResCache bool) (r map[string]*serverpb.LoadReporterResponse, err error) {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name":    "GetServers",
		"serviceAddrs": serviceAddrs,
	})
	lcm.serviceAddrs = serviceAddrs
	r = make(map[string]*serverpb.LoadReporterResponse)

	//var wg sync.WaitGroup
	rch := make(chan map[string]*serverpb.LoadReporterResponse, len(serviceAddrs))
	errch := make(chan *loaderror)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	var count = 0
	var rcount = 0

	for i := range serviceAddrs {
		addr := serviceAddrs[i]
		l, ok := lcm.LoadReporterResList.Load(addr)
		if ok && useResCache {
			r[addr] = l.(*serverpb.LoadReporterResponse)
			logEntry.Info("cache!")
		} else {
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
				//TODO error handule and delete cache
				logEntry.Errorln(e.err)
				lcm.DeleteCache(e.serviceAddr)
				return nil, e.err
			case <-ctx.Done():
				//TODO error handule and delete cache
				logEntry.Error("timeout")
				return nil, nil
			case rll := <-rch:
				for k, v := range rll {
					fmt.Printf("%s \n", k)
					fmt.Printf("%d \n", v.CurLoad)
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

func (lcm *LoadClientMgr) upadteLoad() {

}

func (lcm *LoadClientMgr) updateloop() {

}
