package load_reporter

import (
	"fmt"
	"testing"
	"time"
)

func Test_GetServers(t *testing.T) {
	target := "loginserver,master"

	serviceAddrs := []string{"192.168.171.128:9181", "192.168.171.128:9281", "192.168.171.128:9381"}
	loadCacheInterval := 100 * time.Millisecond
	getServerTimeout := 100 * time.Millisecond

	loadClientMgr := NewLoadClientMgr(target,loadCacheInterval,getServerTimeout)
	r, err := loadClientMgr.GetServers(serviceAddrs, true)
	if err != nil {
		t.Fatal("Get Servers Error: ", err)
	}
	for k, v := range r {
		fmt.Printf("%s \n", k)
		fmt.Printf("%+v \n", v)
	}
	serviceAddrs = []string{"192.168.171.128:9181"}
	_, err = loadClientMgr.GetServers(serviceAddrs, true)
	if err != nil {
		t.Fatal("Get Servers Error: ", err)
	}

}
