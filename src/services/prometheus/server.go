package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

type Prometheus struct {
	reg     *prometheus.Registry
	handler http.Handler
}

func Run(processMetrics bool, goMetrics bool, collector prometheus.Collector) (s *Prometheus) {
	s = &Prometheus{}
	s.reg = prometheus.NewPedanticRegistry()

	if processMetrics {
		s.reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	}
	if goMetrics {
		s.reg.MustRegister(collectors.NewGoCollector())
	}

	if collector != nil {
		s.reg.MustRegister(collector)
	}

	s.handler = promhttp.HandlerFor(s.reg, promhttp.HandlerOpts{})
	return
}

// ServeHTTP implements WebServed interface
func (s *Prometheus) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	s.handler.ServeHTTP(res, req)
}

// URLPatters implements WebServed interface
func (s *Prometheus) URLPatters() (p []string) {
	return []string{"/metrics"}
}
