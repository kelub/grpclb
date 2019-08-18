// Service Discovry

package discovry

type Discovry interface {
	// Name -> Addr    //like DNS
	// name servername
	// tag groupname
	NameResolve(name string, tag string) (Addr string)
}
