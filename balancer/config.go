package balancer

import "time"

type Config struct {
	Balancer struct{
		// 刷新负载值时间间隔，默认5s
		refreshInterval time.Duration
	}
	Discovry struct{
		// 服务发现 consul 地址,默认 .:8500
		consulAddr string
		// resolve等待时间 默认 100ms
		resolveWaitTime time.Duration
	}
	Service struct{
		// 得到 一组服务超时时间，默认 100 ms
		getServerTimeout time.Duration
		// 负载值缓存最大时间  30 * time.Second
		loadCacheInterval  time.Duration

	}
}

func DefaultConfig() *Config{
	return &Config{
		Balancer: struct{ refreshInterval time.Duration }{refreshInterval: 5 * time.Second},
		Discovry: struct {
			consulAddr      string
			resolveWaitTime time.Duration
		}{consulAddr: ":8500", resolveWaitTime: 100 * time.Millisecond},
		Service: struct {
			getServerTimeout  time.Duration
			loadCacheInterval time.Duration
		}{getServerTimeout: 100 * time.Millisecond, loadCacheInterval: 30 * time.Second},
	}
}