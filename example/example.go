package example

type Options struct {
	ServerName    string `flag:"server_name"`
	RPCAddress    string `flag:"rpc-address"`
	RPCPort       int    `flag:"rpc-port"`
	ConsulAddress string `flag:"tcp-port"`
	HealthPort    int    `flag:"HealthPort"`
	ProfPort      int    `flag:"prof_port"`
}
