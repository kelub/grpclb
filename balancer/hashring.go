package balancer

import "sync"

type HashRinger interface {
	AddNode(nodeKey string) error
	RemoveNode(nodeKey string) error
	GetKeyNode(key string) (string, error)
}

type HashRing struct {
	nodes []node
	mu    sync.Mutex
}

type node struct {
	Key string
}
