package dashboard

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/localrivet/wilduri"
)

// RegisterAPI registers the JSON API endpoints on the given router
func RegisterAPI(router *Router) {
	router.Handle("GET /api/v1/metrics/services", apiServicesHandler)
	router.Handle("GET /api/v1/metrics/history/{service}", apiHistoryHandler)
	router.Handle("GET /api/v1/metrics/stats/{service}", apiStatsHandler)
	router.Handle("GET /api/v1/metrics/current", apiCurrentHandler)
}

func apiServicesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if dashboardCollector == nil {
		http.Error(w, `{"error":"Metrics collector not initialized"}`, http.StatusInternalServerError)
		return
	}
	services, err := dashboardCollector.GetServicesWithMetrics()
	if err != nil {
		http.Error(w, `{"error":"Failed to get services"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(services)
}

func apiHistoryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if dashboardCollector == nil {
		http.Error(w, `{"error":"Metrics collector not initialized"}`, http.StatusInternalServerError)
		return
	}
	params := wilduri.GetParams(r)
	service := wilduri.GetString(params, "service", "")
	if service == "" {
		http.Error(w, `{"error":"Service required"}`, http.StatusBadRequest)
		return
	}
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}
	search := strings.ToLower(r.URL.Query().Get("search"))
	recs, err := dashboardCollector.GetDeploymentRecords(service, 1000, 0)
	if err != nil {
		http.Error(w, `{"error":"Failed to get records"}`, http.StatusInternalServerError)
		return
	}
	// Filter by search
	if search != "" {
		filtered := recs[:0]
		for _, rec := range recs {
			if strings.Contains(strings.ToLower(rec.ServiceName), search) ||
				strings.Contains(strings.ToLower(rec.Version), search) ||
				strings.Contains(strings.ToLower(rec.FailureReason), search) {
				filtered = append(filtered, rec)
			}
		}
		recs = filtered
	}
	// Pagination
	end := offset + limit
	if end > len(recs) {
		end = len(recs)
	}
	if offset < len(recs) {
		recs = recs[offset:end]
	} else {
		recs = nil
	}
	json.NewEncoder(w).Encode(recs)
}

func apiStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if dashboardCollector == nil {
		http.Error(w, `{"error":"Metrics collector not initialized"}`, http.StatusInternalServerError)
		return
	}
	params := wilduri.GetParams(r)
	service := wilduri.GetString(params, "service", "")
	if service == "" {
		http.Error(w, `{"error":"Service required"}`, http.StatusBadRequest)
		return
	}
	successRate, _ := dashboardCollector.GetSuccessRate(service)
	avgTime, _ := dashboardCollector.GetAverageDeploymentTime(service)
	rollbacks, _ := dashboardCollector.GetRollbackCount(service)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":      service,
		"success_rate": successRate,
		"avg_time":     avgTime.Seconds(),
		"rollbacks":    rollbacks,
	})
}

func apiCurrentHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if dashboardCollector == nil {
		http.Error(w, `{"error":"Metrics collector not initialized"}`, http.StatusInternalServerError)
		return
	}
	services, err := dashboardCollector.GetServicesWithMetrics()
	if err != nil {
		http.Error(w, `{"error":"Failed to get services"}`, http.StatusInternalServerError)
		return
	}
	status := make(map[string]interface{})
	for _, svc := range services {
		recs, err := dashboardCollector.GetDeploymentRecords(svc, 1, 0)
		if err == nil && len(recs) > 0 {
			status[svc] = recs[0]
		}
	}
	json.NewEncoder(w).Encode(status)
}
