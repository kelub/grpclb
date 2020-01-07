package balancer

import "time"

type Config struct {
	Balancer struct {
		// 刷新负载值时间间隔，默认5s
		refreshInterval time.Duration
	}
	Discovry struct {
		// 服务发现 consul 地址,默认 .:8500
		consulAddr string
		// resolve等待时间 默认 100ms
		resolveWaitTime time.Duration

		// serviceStrategy 服务路由策略kv 路径 默认规则: Router/Strategy
		// 例如获取 target 的负载均衡策略 key: Router/Strategy/<target>
		serviceStrategy string

		// 更新服务地址时间间隔 默认5s
		discovryInterval time.Duration
	}
	Service struct {
		// 得到 一组服务超时时间，默认 100 ms
		getServerTimeout time.Duration
		//// 负载值缓存最大时间  30 * time.Second
		//loadCacheInterval time.Duration

		// 刷新负载值时间间隔，默认5s
		refreshInterval time.Duration
	}
}

func DefaultConfig() *Config {
	return &Config{
		Balancer: struct{ refreshInterval time.Duration }{refreshInterval: 5 * time.Second},
		Discovry: struct {
			consulAddr       string
			resolveWaitTime  time.Duration
			serviceStrategy  string
			discovryInterval time.Duration
		}{consulAddr: ":8500", resolveWaitTime: 100 * time.Millisecond, serviceStrategy: "Router/Strategy", discovryInterval: 5 * time.Second},
		Service: struct {
			getServerTimeout  time.Duration
			loadCacheInterval time.Duration
		}{getServerTimeout: 100 * time.Millisecond, loadCacheInterval: 30 * time.Second},
	}
}
