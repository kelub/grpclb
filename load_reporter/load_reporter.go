// load_reporter
// 负载 grpc server 实现
package load_reporter

import (
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	serverpb "kelub/grpclb/pb/server"
)

type CurLoader interface {
	SetCurLoad(int64)
	GetCurLoad() int64

	SetState(serverpb.ServiceStats)
	GetState() serverpb.ServiceStats
}

type LoadBlancerReporter struct {
	svrLoad CurLoader // main logic mgr Load and State
	curLoad int64     //service cur load value
	stats   serverpb.ServiceStats
}

// LoadReporter 负载报告实现
func (r *LoadBlancerReporter) LoadReporter(ctx context.Context, request *serverpb.LoadReporterRequest) (
	response *serverpb.LoadReporterResponse, err error) {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name": "LoadReporter",
	})
	curLoad := r.svrLoad.GetCurLoad()
	serviceState := r.svrLoad.GetState()
	logEntry.Infof("Load:[%d] State:[%s]", curLoad, serverpb.ServiceStats_name[int32(serviceState)])
	response = &serverpb.LoadReporterResponse{
		CurLoad: curLoad,
		State:   serviceState,
	}
	return response, nil
}

// 注册 grpc 服务
func RegisterLBReporter(g *grpc.Server, lbr *LoadBlancerReporter, svrLoad CurLoader) error {
	logrus.Infof("Register Load Blancer Reporter")
	lbr = &LoadBlancerReporter{
		svrLoad: svrLoad,
	}
	curLoad := svrLoad.GetCurLoad()
	serviceState := svrLoad.GetState()
	logrus.Infof("init Load[%d] init State[%s]", curLoad, serverpb.ServiceStats_name[int32(serviceState)])
	serverpb.RegisterLoadReporterServiceServer(g, lbr)
	return nil
}
