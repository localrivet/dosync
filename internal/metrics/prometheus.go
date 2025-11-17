package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// PrometheusExporter exposes metrics in Prometheus format
type PrometheusExporter struct {
	collector MetricsCollector
	mu        sync.RWMutex
	// Cached metrics to avoid recalculation on every scrape
	lastUpdate time.Time
	cached     string
	cacheTTL   time.Duration
}

// NewPrometheusExporter creates a new Prometheus exporter
func NewPrometheusExporter(collector MetricsCollector) *PrometheusExporter {
	return &PrometheusExporter{
		collector: collector,
		cacheTTL:  10 * time.Second, // Cache metrics for 10 seconds
	}
}

// ServeHTTP implements http.Handler for the /metrics endpoint
func (e *PrometheusExporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.mu.RLock()
	if time.Since(e.lastUpdate) < e.cacheTTL && e.cached != "" {
		cached := e.cached
		e.mu.RUnlock()
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		fmt.Fprint(w, cached)
		return
	}
	e.mu.RUnlock()

	// Generate fresh metrics
	metrics := e.generateMetrics()

	// Update cache
	e.mu.Lock()
	e.cached = metrics
	e.lastUpdate = time.Now()
	e.mu.Unlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprint(w, metrics)
}

// generateMetrics generates all Prometheus metrics
func (e *PrometheusExporter) generateMetrics() string {
	var output string

	// Get all services
	services, err := e.collector.GetServicesWithMetrics()
	if err != nil {
		return "# Error retrieving metrics\n"
	}

	// Initialize metric maps
	deploymentsTotal := make(map[string]map[string]int) // service -> status -> count
	deploymentDuration := make(map[string][]float64)    // service -> durations
	healthCheckFailures := make(map[string]int)         // service -> count
	currentVersion := make(map[string]string)           // service -> version
	rollbacksTotal := make(map[string]int)              // service -> count

	// Collect data for each service
	for _, service := range services {
		records, err := e.collector.GetDeploymentRecords(service, 1000, 0)
		if err != nil {
			continue
		}

		if deploymentsTotal[service] == nil {
			deploymentsTotal[service] = make(map[string]int)
		}

		for _, rec := range records {
			// Count deployments by status
			if rec.Success {
				deploymentsTotal[service]["success"]++
				deploymentDuration[service] = append(deploymentDuration[service], rec.Duration.Seconds())
			} else {
				deploymentsTotal[service]["failed"]++
				healthCheckFailures[service]++
			}

			// Track rollbacks
			if rec.Rollback {
				rollbacksTotal[service]++
			}

			// Track current (most recent) version
			if currentVersion[service] == "" || rec.EndTime.After(time.Time{}) {
				currentVersion[service] = rec.Version
			}
		}
	}

	// Generate HELP and TYPE comments
	output += "# HELP dosync_deployments_total Total number of deployments by service and status\n"
	output += "# TYPE dosync_deployments_total counter\n"
	for service, statuses := range deploymentsTotal {
		for status, count := range statuses {
			output += fmt.Sprintf("dosync_deployments_total{service=\"%s\",status=\"%s\"} %d\n", service, status, count)
		}
	}

	output += "# HELP dosync_deployment_duration_seconds Deployment duration in seconds\n"
	output += "# TYPE dosync_deployment_duration_seconds histogram\n"
	for service, durations := range deploymentDuration {
		if len(durations) == 0 {
			continue
		}
		// Calculate histogram buckets (0.5s, 1s, 5s, 10s, 30s, 60s, 300s, +Inf)
		buckets := []float64{0.5, 1, 5, 10, 30, 60, 300}
		counts := make([]int, len(buckets)+1)
		sum := 0.0

		for _, d := range durations {
			sum += d
			for i, bucket := range buckets {
				if d <= bucket {
					counts[i]++
					break
				}
			}
			counts[len(counts)-1]++ // +Inf bucket
		}

		// Output histogram
		for i, bucket := range buckets {
			output += fmt.Sprintf("dosync_deployment_duration_seconds_bucket{service=\"%s\",le=\"%.1f\"} %d\n",
				service, bucket, counts[i])
		}
		output += fmt.Sprintf("dosync_deployment_duration_seconds_bucket{service=\"%s\",le=\"+Inf\"} %d\n",
			service, counts[len(counts)-1])
		output += fmt.Sprintf("dosync_deployment_duration_seconds_sum{service=\"%s\"} %.2f\n", service, sum)
		output += fmt.Sprintf("dosync_deployment_duration_seconds_count{service=\"%s\"} %d\n", service, len(durations))
	}

	output += "# HELP dosync_health_check_failures_total Total number of health check failures\n"
	output += "# TYPE dosync_health_check_failures_total counter\n"
	for service, count := range healthCheckFailures {
		output += fmt.Sprintf("dosync_health_check_failures_total{service=\"%s\"} %d\n", service, count)
	}

	output += "# HELP dosync_current_version Current deployed version of service\n"
	output += "# TYPE dosync_current_version gauge\n"
	for service, version := range currentVersion {
		output += fmt.Sprintf("dosync_current_version{service=\"%s\",tag=\"%s\"} 1\n", service, version)
	}

	output += "# HELP dosync_rollbacks_total Total number of rollbacks\n"
	output += "# TYPE dosync_rollbacks_total counter\n"
	for service, count := range rollbacksTotal {
		output += fmt.Sprintf("dosync_rollbacks_total{service=\"%s\"} %d\n", service, count)
	}

	return output
}
