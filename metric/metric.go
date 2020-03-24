package metric

import (
	"net/http"

	"github.com/health-monitor/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registry *prometheus.Registry

	isDockerRunning = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "is_docker_running",
			Help: "True/False Docker running.",
		},
	)
	isKubeletRunning = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "is_kubelet_running",
			Help: "True/False Prometheus Kubelet running.",
		},
	)
	restartCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "restart_count",
			Help: "Count of all restart.",
		},
	)
)

func init() {
	registry := prometheus.NewRegistry()
	registry.MustRegister(isDockerRunning)
	registry.MustRegister(isKubeletRunning)
	registry.MustRegister(restartCount)
}

// SetDockerMetric deletes all metrics in this vector.
func SetDockerMetric(val float64) {
	isDockerRunning.Set(val)
}

// SetKubeletMetric deletes all metrics in this vector.
func SetKubeletMetric(val float64) {
	isKubeletRunning.Set(val)
}

// IncrementRestartCount deletes all metrics in this vector.
func IncrementRestartCount() {
	restartCount.Inc()
}

// Run expose metrics to prometheus.
func Run() {
	go func() {
		handler := http.NewServeMux()
		handler.	Handle("/metrics", promhttp.Handler())
		logger.Err(http.ListenAndServe(":2112", nil).Error())
	}()
}
