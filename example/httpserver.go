package example

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"net/http"
)

type HttpServer struct {
	opts *Options
}

func CreateHttpServer() *HttpServer {
	return &HttpServer{}
}

func (h *HttpServer) Main(opts *Options) error {
	h.opts = opts
	logEntry := logrus.WithFields(logrus.Fields{
		"func_name": "httpServer.Main",
		"HealthPort":      h.opts.HealthPort,
	})
	h.HttpFunc()
	logEntry.Infof("start http health check")
	if err := http.ListenAndServe(fmt.Sprintf(":%d", h.opts.HealthPort), nil); err != nil {
		logrus.Panicf("health http listen error: port=%d", h.opts.HealthPort)
		return err
	}
	return nil
}

func (h *HttpServer) HttpFunc() {
	http.HandleFunc("/status", h.statusHandler)
}

func (h *HttpServer) statusHandler(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, "status OK!")
}
