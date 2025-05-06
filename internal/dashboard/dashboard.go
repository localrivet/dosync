package dashboard

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"

	"dosync/internal/config"
	"dosync/internal/metrics"

	"embed"

	"github.com/localrivet/wilduri"
)

var (
	dashboardCollector metrics.MetricsCollector
	dashboardTemplate  *template.Template
	metricsSummaryTmpl *template.Template
	historyRowsTmpl    *template.Template
	serviceOptionsTmpl *template.Template
)

//go:embed templates/*.html
var dashboardTemplates embed.FS

// InitDashboard must be called before using dashboard handlers
func InitDashboard() {
	dashboardTemplate = template.Must(template.New("dashboard").ParseFS(dashboardTemplates, "templates/dashboard.html"))
	metricsSummaryTmpl = template.Must(template.New("metrics_summary").Funcs(template.FuncMap{"mul": func(a, b float64) float64 { return a * b }}).ParseFS(dashboardTemplates, "templates/metrics_summary.html"))
	historyRowsTmpl = template.Must(template.New("history_rows").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"divCeil": func(a, b int) int {
			if b == 0 {
				return 0
			}
			q, r := a/b, a%b
			if r > 0 {
				return q + 1
			}
			return q
		},
	}).ParseFS(dashboardTemplates, "templates/history_rows.html"))
	serviceOptionsTmpl = template.Must(template.New("service_options").ParseFS(dashboardTemplates, "templates/service_options.html"))
}

// Basic Auth middleware
func basicAuth(cfg config.DashboardConfig, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != cfg.User || p != cfg.Pass {
			w.Header().Set("WWW-Authenticate", `Basic realm=\"Restricted\"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// IP Whitelist middleware
func ipWhitelist(cfg config.DashboardConfig, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.IPWhitelist != "" {
			remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
			if remoteIP != cfg.IPWhitelist {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}
		next(w, r)
	}
}

// Router using wilduri
// pattern: e.g. "GET /dashboard", "POST /api/metrics", "GET /api/metrics/{service}"
type Router struct {
	routes map[string]http.HandlerFunc
}

func NewRouter() *Router {
	return &Router{routes: make(map[string]http.HandlerFunc)}
}

func (r *Router) Handle(pattern string, handler http.HandlerFunc) {
	r.routes[pattern] = handler
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	method := req.Method
	for pattern, handler := range r.routes {
		// Split pattern into method and route
		patMethod := "GET"
		patRoute := pattern
		if sp := strings.Index(pattern, " "); sp > 0 {
			patMethod = pattern[:sp]
			patRoute = pattern[sp+1:]
		}
		if method != patMethod {
			continue
		}
		tmpl, err := wilduri.New(patRoute)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid route pattern: %v", err), http.StatusInternalServerError)
			return
		}
		if params, matched := tmpl.Match(path); matched {
			ctx := req.Context()
			for k, v := range params {
				ctx = context.WithValue(ctx, k, v)
			}
			handler(w, req.WithContext(ctx))
			return
		}
	}
	http.NotFound(w, req)
}

// Dashboard page handler
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	dashboardTemplate.Execute(w, nil)
}

// Real metrics API handler (returns HTML for htmx)
func metricsAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if dashboardCollector == nil {
		http.Error(w, "Metrics collector not initialized", http.StatusInternalServerError)
		return
	}
	services, err := dashboardCollector.GetServicesWithMetrics()
	if err != nil {
		http.Error(w, "Failed to get services", http.StatusInternalServerError)
		return
	}
	totalDeployments := 0
	totalRollbacks := 0
	totalSuccess := 0
	totalDuration := int64(0)
	totalRecords := 0
	for _, svc := range services {
		records, err := dashboardCollector.GetDeploymentRecords(svc, 1000, 0)
		if err != nil {
			continue
		}
		totalDeployments += len(records)
		for _, rec := range records {
			if rec.Success {
				totalSuccess++
				totalDuration += int64(rec.Duration)
			}
			if rec.Rollback {
				totalRollbacks++
			}
			totalRecords++
		}
	}
	avgTime := "--"
	if totalSuccess > 0 {
		avgTime = fmt.Sprintf("%.1fs", float64(totalDuration)/float64(totalSuccess)/1e9)
	}
	successRate := 0.0
	if totalRecords > 0 {
		successRate = float64(totalSuccess) / float64(totalRecords)
	}
	metricsSummaryTmpl.Execute(w, map[string]interface{}{
		"Deployments": totalDeployments,
		"SuccessRate": successRate,
		"AvgTime":     avgTime,
		"Rollbacks":   totalRollbacks,
	})
}

// Helper to format duration as human-readable string
func humanDuration(dur metrics.DeploymentRecord) string {
	d := dur.Duration
	if d == 0 {
		return "--"
	}
	sec := int(d.Seconds())
	if sec < 60 {
		return fmt.Sprintf("%ds", sec)
	}
	min := sec / 60
	sec = sec % 60
	return fmt.Sprintf("%dm %02ds", min, sec)
}

// Real deployment history API handler (returns HTML for htmx)
func historyAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if dashboardCollector == nil {
		http.Error(w, "Metrics collector not initialized", http.StatusInternalServerError)
		return
	}
	service := r.URL.Query().Get("service")
	search := strings.ToLower(r.URL.Query().Get("search"))
	sortBy := r.URL.Query().Get("sort")
	page := 1
	limit := 50
	if p := r.URL.Query().Get("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
		if page < 1 {
			page = 1
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
		if limit < 1 {
			limit = 50
		}
	}
	offset := (page - 1) * limit

	type pageData struct {
		Records []metrics.DeploymentRecord
		Page    int
		HasPrev bool
		HasNext bool
		Total   int
	}

	var all []metrics.DeploymentRecord
	if service == "" {
		services, err := dashboardCollector.GetServicesWithMetrics()
		if err != nil || len(services) == 0 {
			historyRowsTmpl.Execute(w, pageData{Records: []metrics.DeploymentRecord{}, Page: page, HasPrev: false, HasNext: false, Total: 0})
			return
		}
		for _, svc := range services {
			recs, err := dashboardCollector.GetDeploymentRecords(svc, 1000, 0)
			if err == nil {
				all = append(all, recs...)
			}
		}
	} else {
		recs, err := dashboardCollector.GetDeploymentRecords(service, 1000, 0)
		if err == nil {
			all = append(all, recs...)
		}
	}

	// Filter by search
	if search != "" {
		filtered := all[:0]
		for _, rec := range all {
			if strings.Contains(strings.ToLower(rec.ServiceName), search) ||
				strings.Contains(strings.ToLower(rec.Version), search) ||
				strings.Contains(strings.ToLower(rec.FailureReason), search) {
				filtered = append(filtered, rec)
			}
		}
		all = filtered
	}

	// Sort
	switch sortBy {
	case "service":
		sort.Slice(all, func(i, j int) bool { return all[i].ServiceName < all[j].ServiceName })
	case "version":
		sort.Slice(all, func(i, j int) bool { return all[i].Version < all[j].Version })
	case "start_time":
		sort.Slice(all, func(i, j int) bool { return all[i].StartTime.Before(all[j].StartTime) })
	case "end_time":
		sort.Slice(all, func(i, j int) bool { return all[i].EndTime.Before(all[j].EndTime) })
	case "success":
		sort.Slice(all, func(i, j int) bool { return !all[i].Success && all[j].Success })
	case "duration":
		sort.Slice(all, func(i, j int) bool { return all[i].Duration < all[j].Duration })
	case "rollback":
		sort.Slice(all, func(i, j int) bool { return !all[i].Rollback && all[j].Rollback })
	default:
		// Default: sort by StartTime descending
		sort.Slice(all, func(i, j int) bool { return all[i].StartTime.After(all[j].StartTime) })
	}

	total := len(all)
	end := offset + limit
	if end > total {
		end = total
	}
	pageRecords := []metrics.DeploymentRecord{}
	if offset < total {
		pageRecords = all[offset:end]
	}
	for i := range pageRecords {
		pageRecords[i].DurationStr = humanDuration(pageRecords[i])
	}
	data := pageData{
		Records: pageRecords,
		Page:    page,
		HasPrev: page > 1,
		HasNext: end < total,
		Total:   total,
	}
	if err := historyRowsTmpl.Execute(w, data); err != nil {
		log.Printf("historyRowsTmpl.Execute error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Service options API handler (returns <option> elements for htmx)
func serviceOptionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if dashboardCollector == nil {
		http.Error(w, "Metrics collector not initialized", http.StatusInternalServerError)
		return
	}
	services, err := dashboardCollector.GetServicesWithMetrics()
	if err != nil {
		serviceOptionsTmpl.Execute(w, []string{})
		return
	}
	serviceOptionsTmpl.Execute(w, services)
}

// StartDashboard starts the dashboard server with the given config and metrics collector
func StartDashboard(cfg config.DashboardConfig, collector *metrics.Collector) {
	dashboardCollector = collector
	if !cfg.Enabled {
		log.Println("Dashboard is disabled in config")
		return
	}
	if cfg.User == "" || cfg.Pass == "" {
		log.Fatal("dashboard.user and dashboard.pass must be set in config")
	}
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	router := NewRouter()

	// Register routes with middleware
	router.Handle("GET /dashboard", ipWhitelist(cfg, basicAuth(cfg, dashboardHandler)))
	router.Handle("GET /api/metrics", ipWhitelist(cfg, basicAuth(cfg, metricsAPIHandler)))
	router.Handle("GET /api/history", ipWhitelist(cfg, basicAuth(cfg, historyAPIHandler)))
	router.Handle("GET /api/services", ipWhitelist(cfg, basicAuth(cfg, serviceOptionsHandler)))
	// Example: router.Handle("POST /api/metrics", somePostHandler)

	addr := ":" + port
	log.Printf("Dashboard running at http://localhost%s/dashboard", addr)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}
	log.Fatal(server.ListenAndServe())
}
