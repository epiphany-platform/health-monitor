package main

import (
	"net/http"

	"github.com/heptiolabs/healthcheck"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Create a new Prometheus registry (you'd likely already have one of these).
	registry := prometheus.NewRegistry()

	// Create a metrics-exposing Handler for the Prometheus registry
	// The healthcheck related metrics will be prefixed with the provided namespace
	health := healthcheck.NewMetricsHandler(registry, "healthd")

	// Add a liveness check that always succeeds
	health.AddLivenessCheck("successful-check", func() error {
		return healthcheck.GoroutineCountCheck(100)()
	})

	// Create an "admin" listener on 0.0.0.0:8080
	adminMux := http.NewServeMux()

	// Expose prometheus metrics on /metrics
	adminMux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	// Expose a liveness check on /live
	adminMux.HandleFunc("/live", health.LiveEndpoint)
	http.ListenAndServe("0.0.0.0:8080", adminMux)
}