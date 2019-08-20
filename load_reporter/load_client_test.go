package load_reporter

import (
	"fmt"
	"testing"
)

func Test_GetServers(t *testing.T) {
	target := "loginserver,master"

	serviceAddrs := []string{"192.168.171.128:9181","192.168.171.128:9281","192.168.171.128:9381"}
	loadClientMgr := NewLoadClientMgr(target)
	r,err := loadClientMgr.GetServers(serviceAddrs)
	if err != nil{
		t.Fatal("Get Servers Error: ",err)
	}
	for k,v := range r{
		fmt.Printf("%s \n",k)
		fmt.Printf("%+v \n",v)
	}

}
