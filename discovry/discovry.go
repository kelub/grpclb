// Service Discovry

package discovry

type Discovry interface {
	// Name -> Addr    //like DNS
	NameResolve(name string) (Addr string)
}
