package discovry

import (
	"fmt"
	"testing"
	"time"
)

var consulAddr = "127.0.0.1:8500"

func Test_NameResolve(t *testing.T) {
	d, err := NewDiscovry(consulAddr)
	if err != nil {
		t.Fatal(err)
	}
	resolveWaitTime := time.Duration(5 * time.Second)
	addrs, err := d.NameResolve("gateserver", "", resolveWaitTime)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(addrs)
}

func Test_NameResolve_Tags(t *testing.T) {
	d, err := NewDiscovry(consulAddr)
	if err != nil {
		t.Fatal(err)
	}
	resolveWaitTime := time.Duration(5 * time.Second)
	addrs, err := d.NameResolve("loginserver", "B", resolveWaitTime)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(addrs)
}

// 15838233822
