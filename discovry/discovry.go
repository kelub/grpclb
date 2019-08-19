// Service Discovry

package discovry

import (
	"time"
)

type Discovry interface {
	// Name -> Addr    //like DNS
	// name servername
	// tag groupname
	NameResolve(serviceName string, tag string, resolveWaitTime time.Duration) ([]string, error)
}
