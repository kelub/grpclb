package balancer

import (
	"fmt"
	"sync"
	"testing"
)

func TestService_GetServer(t *testing.T) {
	b := NewBalancer()
	tags := []string{"master"}
	serviceName := []string{"loginserver", "gatewayserver", "xx"}
	//ctx ,cancel := context.WithTimeout(context.Background(),10 * time.Second)
	wg := &sync.WaitGroup{}
	for i := 0; i < len(serviceName); i++ {
		s := serviceName[i]
		wg.Add(1)
		go getServers(t, b, wg, s, tags)
	}
	wg.Wait()
	fmt.Print("\n over \n")
	return
}

func getServers(t *testing.T, b *Balancer, wg *sync.WaitGroup, serviceName string, tags []string) {
	r, err := b.GetServers(serviceName, tags)
	defer func() {
		fmt.Println("defer getServers")
		wg.Done()
	}()
	if err != nil {
		fmt.Println("err: ", err)
		t.FailNow()
		return
	}
	fmt.Printf("res:[%v] \n", r)
	return
}
