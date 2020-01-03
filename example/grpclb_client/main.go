package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"kelub/grpclb/example"
	serverpb "kelub/grpclb/pb/server"
	"os"
	"time"
)

var opt example.Options

func init() {
	flag.StringVar(&opt.RPCAddress, "addr", "127.0.0.1", "Server address. Default: 127.0.0.1")
	flag.StringVar(&opt.RPCPort, "prot", "8081", "Server address. Default: 8081")
	flag.StringVar(&opt.ServerName, "name", "A", "Server address. Default: A")
}

func main() {
	flag.Parse()
	cc := GetRPCClient(opt.RPCAddress, opt.RPCPort)
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
		r, err := GetLoad(cc)
		if err != nil {
			continue
		}
		load := r.GetCurLoad()
		state := r.GetState()
		logrus.Infof("load:[%d],state:[%s]", load, serverpb.ServiceStats_name[int32(state)])
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

}

func GetRPCClient(addr string, port string) serverpb.LoadReporterServiceClient {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name": "RunRPCClient",
		"addr":      addr,
		"port":      port,
	})
	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", addr, port), grpc.WithInsecure())
	if err != nil {
		logEntry.Error("did not connect ", err)
	}
	cc := serverpb.NewLoadReporterServiceClient(conn)
	return cc
}

func GetLoad(cc serverpb.LoadReporterServiceClient) (r *serverpb.LoadReporterResponse, err error) {
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name": "GetLoad",
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	LRRRequest := &serverpb.LoadReporterRequest{}
	r, err = cc.LoadReporter(ctx, LRRRequest)
	if err != nil {
		logEntry.Errorf("could not verify: %v", err)
		return nil, err
	}
	return r, nil
}
