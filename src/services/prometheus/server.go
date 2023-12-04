package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

type Collector struct {
	reg     *prometheus.Registry
	handler http.Handler
}

func Run(processMetrics bool, goMetrics bool) (s *Collector) {
	s = &Collector{}
	s.reg = prometheus.NewPedanticRegistry()

	if processMetrics {
		s.reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	}
	if goMetrics {
		s.reg.MustRegister(collectors.NewGoCollector())
	}
	s.handler = promhttp.HandlerFor(s.reg, promhttp.HandlerOpts{})
	return
}
func (s *Collector) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	s.handler.ServeHTTP(res, req)
}

func (s *Collector) UrlPatters() (p []string) {
	return []string{"/metrics"}
}
