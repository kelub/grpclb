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
	logEntry.Infof("%+v",r)
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
	loadClientList      *sync.Map //addr: *LoadClient
	loadReporterResList *sync.Map //addr: *serverpb.LoadReporterResponse
}

func NewLoadClientMgr(target string) *LoadClientMgr {
	lcm := &LoadClientMgr{
		target:         target,
		loadClientList: new(sync.Map),
		loadReporterResList: new(sync.Map),
	}
	return lcm
}

func (lcm *LoadClientMgr) getServer(ctx context.Context,
	stopch chan struct{}, rch chan map[string]*serverpb.LoadReporterResponse,
	lc *LoadClient, serviceAddr string) error {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name":   "getServer",
		"serviceAddr": serviceAddr,
	})
	addrLrr := make(map[string]*serverpb.LoadReporterResponse)
	lrr, err := lc.GetLoad(ctx)
	if err != nil {
		logEntry.Error(err)
		stopch <- struct{}{}
		return err
	}
	lcm.loadReporterResList.Store(serviceAddr, lrr)
	addrLrr[serviceAddr] = lrr
	rch <- addrLrr

	return nil
}

func (lcm *LoadClientMgr) GetServers(serviceAddrs []string) (r map[string]*serverpb.LoadReporterResponse, err error) {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name":    "GetServers",
		"serviceAddrs": serviceAddrs,
	})
	lcm.serviceAddrs = serviceAddrs
	r = make(map[string]*serverpb.LoadReporterResponse)
	//var wg sync.WaitGroup
	rch := make(chan map[string]*serverpb.LoadReporterResponse, len(serviceAddrs))
	stopch := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	for i := range serviceAddrs {
		addr := serviceAddrs[i]
		var lc *LoadClient
		v, ok := lcm.loadClientList.Load(addr)
		if ok {
			lc = v.(*LoadClient)
		} else {
			lc, err = NewLoadClient(ctx, addr)
			if err != nil {
				return nil, err
			}
			lcm.loadClientList.Store(addr, lc)
		}
		go lcm.getServer(ctx, stopch, rch, lc, addr)
	}

	for {
		select {
		case <-stopch:
			//TODO error handule
			return nil, nil
		case <-ctx.Done():
			//TODO error handule
			logEntry.Error("timeout")
			return nil, nil
		case rll := <-rch:
			for k, v := range rll {
				fmt.Printf("%s \n",k)
				fmt.Printf("%+v \n",v)
				r[k] = v
			}
			if len(r) == len(serviceAddrs) {
				return
			}
		}
	}
}
