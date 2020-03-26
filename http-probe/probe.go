package http-probe

import (
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
)

type probeHandler struct {
	handler   Handler
	registry  prometheus.Registerer
	namespace string
}

// NewMetricsHandler returns a healthcheck Handler that also exposes metrics
// into the provided Prometheus registry.
func NewMetricsHandler(registry prometheus.Registerer, namespace string) Handler {
	return &metricsHandler{
		handler:   NewHandler(),
		registry:  registry,
		namespace: namespace,
	}
}

func (h *metricsHandler) AddLivenessCheck(name string, check Check) {
	h.handler.AddLivenessCheck(name, h.wrap(name, check))
}

func (h *metricsHandler) AddReadinessCheck(name string, check Check) {
	h.handler.AddReadinessCheck(name, h.wrap(name, check))
}

func (h *metricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}

func (h *metricsHandler) LiveEndpoint(w http.ResponseWriter, r *http.Request) {
	h.handler.LiveEndpoint(w, r)
}

func (h *metricsHandler) ReadyEndpoint(w http.ResponseWriter, r *http.Request) {
	h.handler.ReadyEndpoint(w, r)
}

func (h *metricsHandler) wrap(name string, check Check) Check {
	h.registry.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace:   h.namespace,
			Subsystem:   "http-probe",
			Name:        "status",
			Help:        "Current check status (0 indicates success, 1 indicates failure)",
			ConstLabels: prometheus.Labels{"check": name},
		},
		func() float64 {
			if check() == nil {
				return 0
			}
			return 1
		},
	))
	return check
}
