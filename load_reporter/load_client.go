package load_reporter

import (
	"context"
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

	serviceAddr string
	load        int64
	state       serverpb.ServiceStats
}

func NewLoadClient(serviceAddr string) (*LoadClient, error) {
	conn, err := grpc.Dial(serviceAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	lrc := serverpb.NewLoadReporterServiceClient(conn)
	return &LoadClient{
		cc:          conn,
		lrc:         lrc,
		serviceAddr: serviceAddr,
	}, nil
}

func (lc *LoadClient) GetLoad() (r *serverpb.LoadReporterResponse, err error) {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name": "GetLoad",
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err = lc.lrc.LoadReporter(ctx, &serverpb.LoadReporterRequest{})
	if err != nil {
		logEntry.Errorf("could not verify: %v", err)
		return nil, err
	}
	return
}

func (lc *LoadClient) GetCurLoad() (int64, error) {
	lrr, err := lc.GetLoad()
	if err != nil {
		return 0, err
	}
	return lrr.GetCurLoad(), nil
}

type LoadClientMgr struct {
	//One group target
	target         string
	serviceAddrs   []string
	loadClientList *sync.Map
}

func NewLoadClientMgr(target string) *LoadClientMgr {
	lcm := &LoadClientMgr{
		target:         target,
		loadClientList: new(sync.Map),
	}
	return lcm
}

func (lcm *LoadClientMgr) GetServer(serviceAddrs string) {

}
