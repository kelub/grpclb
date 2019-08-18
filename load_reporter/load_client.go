package load_reporter

import (
	"google.golang.org/grpc"
	serverpb "kelub/grpclb/pb/server"
)

type LoadClient struct {
	cc  			*grpc.ClientConn
	lrc 			serverpb.LoadReporterServiceClient

	serviceAddr 	string
	load 			int64
	state 			serverpb.ServiceStats

}

func NewLoadClient() *LoadClient{

}

func (lc *LoadClient) GetLoad()(){

}