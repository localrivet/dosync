package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dosync/internal/metrics"
	"net/url"
	"os"
)

func TestMain(m *testing.M) {
	InitDashboard()
	os.Exit(m.Run())
}

type fakeCollector struct {
	records map[string][]metrics.DeploymentRecord
}

func (f *fakeCollector) GetDeploymentRecords(service string, limit, offset int) ([]metrics.DeploymentRecord, error) {
	recs := f.records[service]
	if offset > len(recs) {
		return []metrics.DeploymentRecord{}, nil
	}
	end := offset + limit
	if end > len(recs) {
		end = len(recs)
	}
	return recs[offset:end], nil
}
func (f *fakeCollector) GetServicesWithMetrics() ([]string, error) {
	services := make([]string, 0, len(f.records))
	for k := range f.records {
		services = append(services, k)
	}
	return services, nil
}

// Unused methods for metrics.Collector interface
func (f *fakeCollector) GetSuccessRate(service string) (float64, error) { return 0.75, nil }
func (f *fakeCollector) GetAverageDeploymentTime(service string) (time.Duration, error) {
	return 42 * time.Second, nil
}
func (f *fakeCollector) GetRollbackCount(service string) (int, error)                { return 2, nil }
func (f *fakeCollector) RecordDeploymentStart(string, string) error                  { return nil }
func (f *fakeCollector) RecordDeploymentSuccess(string, string, time.Duration) error { return nil }
func (f *fakeCollector) RecordDeploymentFailure(string, string, string) error        { return nil }
func (f *fakeCollector) RecordRollback(string, string, string) error                 { return nil }
func (f *fakeCollector) UpdateRetentionConfig(metrics.RetentionConfig) error         { return nil }
func (f *fakeCollector) RunRetentionNow() (map[string]int64, error)                  { return nil, nil }
func (f *fakeCollector) Close() error                                                { return nil }

func TestHumanDuration(t *testing.T) {
	cases := []struct {
		dur      time.Duration
		expected string
	}{
		{0, "--"},
		{45 * time.Second, "45s"},
		{90 * time.Second, "1m 30s"},
		{3600 * time.Second, "60m 00s"},
	}
	for _, c := range cases {
		rec := metrics.DeploymentRecord{Duration: c.dur}
		got := humanDuration(rec)
		if got != c.expected {
			t.Errorf("humanDuration(%v) = %q, want %q", c.dur, got, c.expected)
		}
	}
}

func TestHistoryAPIHandler_Empty(t *testing.T) {
	dashboardCollector = &fakeCollector{records: map[string][]metrics.DeploymentRecord{}}
	r := httptest.NewRequest("GET", "/api/history", nil)
	w := httptest.NewRecorder()
	historyAPIHandler(w, r)
	resp := w.Body.String()
	if !strings.Contains(resp, "No deployment records found") {
		t.Errorf("expected empty state message, got: %s", resp)
	}
}

func TestHistoryAPIHandler_SingleRecord(t *testing.T) {
	dashboardCollector = &fakeCollector{records: map[string][]metrics.DeploymentRecord{
		"svc": {
			{
				ServiceName:   "svc",
				Version:       "v1",
				StartTime:     time.Now().Add(-2 * time.Minute),
				EndTime:       time.Now().Add(-1 * time.Minute),
				Success:       false,
				Duration:      45 * time.Second,
				FailureReason: "bad config",
				DurationStr:   "45s",
			},
		},
	}}
	r := httptest.NewRequest("GET", "/api/history?service=svc", nil)
	w := httptest.NewRecorder()
	historyAPIHandler(w, r)
	resp := w.Body.String()
	if !strings.Contains(resp, "svc") || !strings.Contains(resp, "45s") {
		t.Errorf("expected record in output, got: %s", resp)
	}
	if !strings.Contains(resp, "⚠️") || !strings.Contains(resp, "bad config") {
		t.Errorf("expected failure reason tooltip, got: %s", resp)
	}
}

func TestHistoryAPIHandler_Pagination(t *testing.T) {
	recs := make([]metrics.DeploymentRecord, 0, 60)
	for i := 0; i < 60; i++ {
		recs = append(recs, metrics.DeploymentRecord{
			ServiceName: "svc",
			Version:     fmt.Sprintf("v%d", i),
			StartTime:   time.Now().Add(-time.Duration(i) * time.Minute),
			EndTime:     time.Now().Add(-time.Duration(i-1) * time.Minute),
			Success:     true,
			Duration:    90 * time.Second,
			DurationStr: "1m 30s",
		})
	}
	dashboardCollector = &fakeCollector{records: map[string][]metrics.DeploymentRecord{"svc": recs}}

	r := httptest.NewRequest("GET", "/api/history?service=svc&page=2", nil)
	w := httptest.NewRecorder()
	historyAPIHandler(w, r)
	resp := w.Body.String()
	if !strings.Contains(resp, "Page 2") {
		t.Errorf("expected page 2 in output, got: %s", resp)
	}
	if !strings.Contains(resp, "1m 30s") {
		t.Errorf("expected human-readable duration, got: %s", resp)
	}
}

func TestHistoryAPIHandler_SearchAndSort(t *testing.T) {
	recs := []metrics.DeploymentRecord{
		{ServiceName: "svcA", Version: "v1", StartTime: time.Now().Add(-3 * time.Hour), Success: true, Duration: 30 * time.Second, FailureReason: "", Rollback: false, DurationStr: "30s"},
		{ServiceName: "svcB", Version: "v2", StartTime: time.Now().Add(-2 * time.Hour), Success: false, Duration: 45 * time.Second, FailureReason: "bad config", Rollback: true, DurationStr: "45s"},
		{ServiceName: "svcC", Version: "v3", StartTime: time.Now().Add(-1 * time.Hour), Success: true, Duration: 60 * time.Second, FailureReason: "", Rollback: false, DurationStr: "1m 0s"},
	}
	dashboardCollector = &fakeCollector{records: map[string][]metrics.DeploymentRecord{"svcA": {recs[0]}, "svcB": {recs[1]}, "svcC": {recs[2]}}}

	// Search by service
	r := httptest.NewRequest("GET", "/api/history?search=svcB", nil)
	w := httptest.NewRecorder()
	historyAPIHandler(w, r)
	resp := w.Body.String()
	if !strings.Contains(resp, "svcB") || strings.Contains(resp, "svcA") || strings.Contains(resp, "svcC") {
		t.Errorf("search by service failed, got: %s", resp)
	}

	// Search by version
	r = httptest.NewRequest("GET", "/api/history?search=v3", nil)
	w = httptest.NewRecorder()
	historyAPIHandler(w, r)
	resp = w.Body.String()
	if !strings.Contains(resp, "svcC") || strings.Contains(resp, "svcA") || strings.Contains(resp, "svcB") {
		t.Errorf("search by version failed, got: %s", resp)
	}

	// Search by failure reason
	r = httptest.NewRequest("GET", "/api/history?search="+url.QueryEscape("bad config"), nil)
	w = httptest.NewRecorder()
	historyAPIHandler(w, r)
	resp = w.Body.String()
	if !strings.Contains(resp, "svcB") || strings.Contains(resp, "svcA") || strings.Contains(resp, "svcC") {
		t.Errorf("search by failure reason failed, got: %s", resp)
	}

	// Sort by service
	r = httptest.NewRequest("GET", "/api/history?sort=service", nil)
	w = httptest.NewRecorder()
	historyAPIHandler(w, r)
	resp = w.Body.String()
	firstIdx := strings.Index(resp, "svcA")
	secondIdx := strings.Index(resp, "svcB")
	thirdIdx := strings.Index(resp, "svcC")
	if !(firstIdx < secondIdx && secondIdx < thirdIdx) {
		t.Errorf("sort by service failed, got order: %d %d %d", firstIdx, secondIdx, thirdIdx)
	}

	// Sort by duration
	r = httptest.NewRequest("GET", "/api/history?sort=duration", nil)
	w = httptest.NewRecorder()
	historyAPIHandler(w, r)
	resp = w.Body.String()
	idxA := strings.Index(resp, "svcA")
	idxB := strings.Index(resp, "svcB")
	idxC := strings.Index(resp, "svcC")
	if !(idxA < idxB && idxB < idxC) {
		t.Errorf("sort by duration failed, got order: %d %d %d", idxA, idxB, idxC)
	}
}

func TestAPIServicesHandler(t *testing.T) {
	dashboardCollector = &fakeCollector{records: map[string][]metrics.DeploymentRecord{
		"svc1": {}, "svc2": {},
	}}
	router := NewRouter()
	RegisterAPI(router)
	r := httptest.NewRequest("GET", "/api/v1/metrics/services", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var services []string
	if err := json.Unmarshal(w.Body.Bytes(), &services); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(services) != 2 || services[0] != "svc1" && services[1] != "svc2" && services[0] != "svc2" && services[1] != "svc1" {
		t.Errorf("unexpected services: %v", services)
	}
}

func TestAPICurrentHandler(t *testing.T) {
	dashboardCollector = &fakeCollector{records: map[string][]metrics.DeploymentRecord{
		"svc": {{ServiceName: "svc", Version: "v1"}},
	}}
	router := NewRouter()
	RegisterAPI(router)
	r := httptest.NewRequest("GET", "/api/v1/metrics/current", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var status map[string]metrics.DeploymentRecord
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if rec, ok := status["svc"]; !ok || rec.Version != "v1" {
		t.Errorf("unexpected status: %v", status)
	}
}
