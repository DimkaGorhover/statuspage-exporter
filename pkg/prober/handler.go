package prober

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/sergeyshevch/statuspage-exporter/pkg/engines"
)

func createMetrics() (*prometheus.GaugeVec, *prometheus.GaugeVec, *prometheus.GaugeVec) {
	componentStatus := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "statuspage_component",
			Help: "Status of a service component: " +
				"0 - Unknown, 1 - Operational, 2 - Planned Maintenance, " +
				"3 - Degraded Performance, 4 - Partial Outage, 5 - Major Outage, 6 - Security Issue",
		},
		[]string{"service", "status_page_url", "component"},
	)
	overallStatus := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "statuspage_overall",
			Help: "Overall status of a service: " +
				"0 - Unknown, 1 - Operational, 2 - Planned Maintenance, " +
				"3 - Degraded Performance, 4 - Partial Outage, 5 - Major Outage, 6 - Security Issue",
		},
		[]string{"service", "status_page_url"},
	)
	serviceStatusDurationGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "service_status_fetch_duration_seconds",
			Help: "Returns how long the service status fetch took to complete in seconds",
		},
		[]string{"status_page_url"},
	)

	return componentStatus, overallStatus, serviceStatusDurationGauge
}

// Handler returns a http handler for /probe endpoint.
func Handler(log *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		targetURL := r.URL.Query().Get("target")
		if targetURL == "" {
			http.Error(w, "target is required", http.StatusBadRequest)
			return
		}

		parsedURL, err := url.Parse(targetURL)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to parse target url: %v", err), http.StatusBadRequest)
			return
		}

		if parsedURL.Scheme == "" {
			parsedURL.Scheme = "https"
		}

		targetURL = parsedURL.String()

		componentStatus, overallStatus, serviceStatusDurationGauge := createMetrics()
		registry := prometheus.NewRegistry()
		registry.MustRegister(componentStatus)
		registry.MustRegister(overallStatus)
		registry.MustRegister(serviceStatusDurationGauge)

		err = engines.FetchStatus(log, targetURL, componentStatus, overallStatus)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to fetch status for %s: %v", targetURL, err), http.StatusInternalServerError)
			return
		}

		duration := time.Since(start).Seconds()
		serviceStatusDurationGauge.WithLabelValues(targetURL).Set(duration)

		promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	}
}
