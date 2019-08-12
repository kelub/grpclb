package main

import (
	"flag"
	"fmt"
	"github.com/Sirupsen/logrus"
	"google.golang.org/grpc"
	"kelub/grpclb/example"
	lr "kelub/grpclb/load_reporter"
	serverpb "kelub/grpclb/pb/server"
	"net"
	"sync"
)

var opt example.Options

func main() {
	flag.Parse()
	s := grpc.NewServer()
	lr.RegisterLBReporter(s, &lr.LoadBlancerReporter{}, NewLoadMgr())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		RunRPCServer(s, opt.RPCAddress, opt.RPCPort)
		wg.Done()
	}()
	wg.Add(1)

	go func() {
		Start()
		wg.Done()
	}()
	wg.Wait()
}

func init() {
	flag.StringVar(&opt.RPCAddress, "addr", "127.0.0.1", "Server address. Default: 127.0.0.1")
	flag.StringVar(&opt.RPCPort, "prot", "8081", "Server address. Default: 8081")
	flag.StringVar(&opt.ServerName, "name", "A", "Server address. Default: A")
}

func Start() {
	//dothings
	ch := make(chan struct{})
	<-ch
	return
}

func RunRPCServer(s *grpc.Server, addr string, port string) {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name": "RunRPCServer",
		"addr":      addr,
		"port":      port,
	})
	logEntry.Infoln("RPC Starting...")
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", addr, port))
	if err != nil {
		logEntry.Info(err)
		return
	}
	err = s.Serve(lis)
	if err != nil {
		logEntry.Infoln("启动 RPC 服务失败")
	}
}

type LoadMgr struct {
	curLoad int64
	state   serverpb.ServiceStats
}

func (l *LoadMgr) SetCurLoad(int64) {
	l.curLoad = 100
}

func (l *LoadMgr) GetCurLoad() int64 {
	return l.curLoad
}

func (l *LoadMgr) SetState(serverpb.ServiceStats) {
	l.state = serverpb.ServiceStats_RUN
}

func (l *LoadMgr) GetState() serverpb.ServiceStats {
	return l.state
}

func NewLoadMgr() *LoadMgr {
	return &LoadMgr{
		curLoad: 0,
		state:   serverpb.ServiceStats_STARTING,
	}
}
