package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
	"google.golang.org/grpc"
	"kelub/grpclb/example"
	lr "kelub/grpclb/load_reporter"
	serverpb "kelub/grpclb/pb/server"
	"net"
	"strings"
	"sync"
)

var runningRPC int64 = 0

var opt example.Options
var defaultServiceStrategy = "Router/Strategy"

func main() {
	flag.Parse()
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name": "main",
	})
	logEntry.Infof("%+v", opt)
	rpcOption := make([]grpc.ServerOption, 0)
	rpcOption = append(rpcOption, grpc.UnaryInterceptor(GrpcInterceptor))
	s := grpc.NewServer(rpcOption...)
	lr.RegisterLBReporter(s, &lr.LoadBlancerReporter{}, NewLoadMgr())

	RegisterToConsul()

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
	flag.IntVar(&opt.RPCPort, "port", 8081, "Server address. Default: 8081")
	flag.StringVar(&opt.ServerName, "name", "gateserver", "Server address. Default: gateserver")
	flag.StringVar(&opt.ServerID, "serverid", "9999", "Server address. Default: 9999")
	flag.StringVar(&opt.ServerTags, "tags", "master_1", "service groups. Default: master_1")

	flag.StringVar(&opt.Strategy, "strategy", "0", "service strategy. Default: 0")

	flag.StringVar(&opt.ConsulAddress, "consulAddr", "127.0.0.1", "Server address. Default: 127.0.0.1")
	flag.IntVar(&opt.HealthPort, "healthPort", 8082, "Server HealthPort. Default: 8082")
	flag.IntVar(&opt.ProfPort, "profPort", 8080, "Server ProfPort. Default: 8080")
}

func Start() {
	//dothings
	ch := make(chan struct{})
	<-ch
	return
}

func RunRPCServer(s *grpc.Server, addr string, port int) {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name": "RunRPCServer",
		"addr":      addr,
		"port":      port,
	})
	logEntry.Infoln("RPC Starting...")
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, port))
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

func (l *LoadMgr) SetCurLoad(curLoad int64) {
	l.curLoad = curLoad
}

func (l *LoadMgr) GetCurLoad() int64 {
	return l.curLoad
}

func (l *LoadMgr) SetState(state serverpb.ServiceStats) {
	l.state = state
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

func GrpcInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	runningRPC++
	resp, err = handler(ctx, req)
	runningRPC--
	return resp, err
}

////

func RegisterToConsul() error {
	consulAddr := fmt.Sprintf("%s:8500", opt.RPCAddress)
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name":  "RegisterToConsul",
		"consulAddr": consulAddr,
	})
	conf := consulapi.DefaultConfig()
	conf.Address = consulAddr
	consulClient, err := consulapi.NewClient(conf)
	if err != nil {
		return err
	}
	agent := consulClient.Agent()
	ck := &consulapi.AgentServiceCheck{
		HTTP:                           fmt.Sprintf("http://%s:%d/status", opt.RPCAddress, opt.HealthPort),
		Interval:                       "3s",
		Timeout:                        "5s",
		DeregisterCriticalServiceAfter: "300s",
	}
	//tags_test := []string{"A", "B", "CD", "EFG"}
	opt.ServerTags = strings.Replace(opt.ServerTags, " ", "", -1)
	tags := strings.Split(opt.ServerTags, ",")
	registration := &consulapi.AgentServiceRegistration{
		ID:      opt.ServerID,
		Name:    opt.ServerName,
		Tags:    tags,
		Port:    opt.RPCPort,
		Address: opt.RPCAddress,
		Check:   ck,
	}
	err = agent.ServiceRegister(registration)
	if err != nil {
		return nil
	}

	for _, tag := range tags {
		Pair := &consulapi.KVPair{}
		Pair.Key = defaultServiceStrategy + "/" + opt.ServerName + "/" + tag
		Pair.Value = []byte(opt.Strategy)
		_, err := consulClient.KV().Put(Pair, nil)
		if err != nil {
			return nil
		}
	}

	logEntry.Infof("ServiceRegister")
	httpServer := example.CreateHttpServer()
	go func() {
		httpServer.Main(&opt)
	}()
	return nil
}
