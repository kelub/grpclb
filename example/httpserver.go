package example

import (
	"fmt"
	"net/http"
	"github.com/Sirupsen/logrus"
)

type HttpServer struct {
	opts *Options
}

func CreateHttpServer() *HttpServer{
	return &HttpServer{}
}

func (h *HttpServer)Main(opts *Options) error{
	h.opts = opts
	if err := http.ListenAndServe(fmt.Sprintf(":%d",h.opts.HealthPort),nil);err != nil{
		logrus.Panicf("health http listen error: port=%d", h.opts.HealthPort)
		return err
	}
	return nil
}

func (h *HttpServer)HttpFunc(){
	http.HandleFunc("status",h.statusHandler)
}

func (h *HttpServer)statusHandler(w http.ResponseWriter, _ *http.Response){
	fmt.Fprint(w, "status OK!")
}