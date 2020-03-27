package metric

import (
	"net/http"

	"github.com/healthd/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
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
	prometheus.MustRegister(isDockerRunning)
	prometheus.MustRegister(isKubeletRunning)
	if err := prometheus.Register(restartCount); err != nil {
		logger.Warning(err.Error())
		panic(err)
	}
	isDockerRunning.Set(0)
	isKubeletRunning.Set(0)
	restartCount.Inc()
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
func Run(port *string) {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":"+*port, nil); err != nil {
			logger.Err(err.Error())
			panic(err)
		}
	}()
}
