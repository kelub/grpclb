package discovry

import (
	"kelub/grpclb/example"
	"testing"

	"github.com/hashicorp/consul/api"
)

func Test_RegisterToConsul(t *testing.T) {
	consulAddr := ""
	d, err := NewDiscovry(consulAddr)
	if err != nil {
		t.Fatal(err)
		return
	}
	consulClient := d.consulClient
	agent := consulClient.Agent()
	ck := &api.AgentServiceCheck{
		HTTP:                           "http://127.0.0.1/status",
		Interval:                       "3s",
		Timeout:                        "5s",
		DeregisterCriticalServiceAfter: "300s",
	}
	registration := &api.AgentServiceRegistration{
		ID:   "9999",
		Name: "gateserver",
		//Tags: "",
		Port:    8081,
		Address: "127.0.0.1",
		Check:   ck,
	}
	err = agent.ServiceRegister(registration)
	if err != nil {
		t.Fatal(err)
		return
	}

	httpServer := example.CreateHttpServer()
	opts := &example.Options{
		ServerName    : "gateserver",
		RPCAddress    : "127.0.0.1",
		RPCPort       : "8081",
		ConsulAddress : "127.0.0.1",
		HealthPort    : 8082,
		ProfPort      : 8080,
	}
	go func(){
		httpServer.Main(opts)
	}()
}

// 15838233822
