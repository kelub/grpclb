package balancer

type StrategyID int32

const (
	//负载均衡方式 默认方式 返回最小负载的服务器
	Strategy_LoadBalancer StrategyID = 0
	//轮询方式
	Strategy_RollPoling StrategyID = 1
	//Hash 方式
	Strategy_Hash StrategyID = 2
)

type Strategy struct {
	StrategyID int32
}
