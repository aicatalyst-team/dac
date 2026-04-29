package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/bruin-data/dac/pkg/dashboard"
	"github.com/bruin-data/dac/pkg/query"
)

// ---------------------------------------------------------------------------
// Mock backend
// ---------------------------------------------------------------------------

type mockBackend struct {
	result *query.QueryResult
	err    error
	mu     sync.Mutex
	calls  []mockCall
}

type mockCall struct {
	Connection string
	SQL        string
}

func (m *mockBackend) Execute(_ context.Context, conn string, sql string) (*query.QueryResult, error) {
	m.mu.Lock()
	m.calls = append(m.calls, mockCall{Connection: conn, SQL: sql})
	m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

// ---------------------------------------------------------------------------
// generateSingleMetricSQL unit tests
// ---------------------------------------------------------------------------

func gaSource() *dashboard.Source {
	return &dashboard.Source{
		Table:      "`project.dataset.events_*`",
		DateColumn: "event_date",
		DateFormat: "%Y%m%d",
	}
}

func gaMetrics() map[string]dashboard.Metric {
	return map[string]dashboard.Metric{
		"page_views":        {Aggregate: "count", Filter: map[string]string{"event_name": "page_view"}},
		"users":             {Aggregate: "count_distinct", Column: "user_pseudo_id"},
		"sessions":          {Aggregate: "count", Filter: map[string]string{"event_name": "session_start"}},
		"pages_per_session": {Expression: "page_views / sessions"},
	}
}

func gaSemantic() *dashboard.SemanticLayer {
	return &dashboard.SemanticLayer{
		Source:  gaSource(),
		Metrics: gaMetrics(),
	}
}

func TestFanoutSingleMetric_AggregateMetric(t *testing.T) {
	d := &dashboard.Dashboard{
		Semantic: gaSemantic(),
	}
	merged := &WidgetQueryResult{
		Columns: []struct {
			Name string `json:"name"`
			Type string `json:"type,omitempty"`
		}{
			{Name: "events"}, {Name: "page_views"}, {Name: "sessions"}, {Name: "users"},
		},
		Rows:  [][]any{{5000.0, 1200.0, 400.0, 300.0}},
		Query: "SELECT ...",
	}

	wr := FanoutSingleMetric(merged, "page_views", "SELECT ...", d)
	if wr.Error != "" {
		t.Fatal(wr.Error)
	}
	if len(wr.Columns) != 1 || wr.Columns[0].Name != "page_views" {
		t.Errorf("expected column 'page_views', got %v", wr.Columns)
	}
	if len(wr.Rows) != 1 || wr.Rows[0][0] != 1200.0 {
		t.Errorf("expected value 1200, got %v", wr.Rows)
	}
}

func TestFanoutSingleMetric_ExpressionMetric(t *testing.T) {
	d := &dashboard.Dashboard{
		Semantic: gaSemantic(),
	}
	merged := &WidgetQueryResult{
		Columns: []struct {
			Name string `json:"name"`
			Type string `json:"type,omitempty"`
		}{
			{Name: "events"}, {Name: "page_views"}, {Name: "sessions"}, {Name: "users"},
		},
		Rows:  [][]any{{5000.0, 1000.0, 200.0, 150.0}},
		Query: "SELECT ...",
	}

	wr := FanoutSingleMetric(merged, "pages_per_session", "SELECT ...", d)
	if wr.Error != "" {
		t.Fatal(wr.Error)
	}
	// page_views / sessions = 1000 / 200 = 5.0
	if len(wr.Rows) != 1 {
		t.Fatal("expected 1 row")
	}
	val, ok := wr.Rows[0][0].(float64)
	if !ok || val != 5.0 {
		t.Errorf("expected 5.0, got %v", wr.Rows[0][0])
	}
}

func TestFanoutSingleMetric_Error(t *testing.T) {
	d := &dashboard.Dashboard{
		Semantic: gaSemantic(),
	}
	merged := &WidgetQueryResult{Error: "query failed", Query: "SELECT ..."}

	wr := FanoutSingleMetric(merged, "page_views", "SELECT ...", d)
	if wr.Error == "" {
		t.Fatal("expected error to propagate")
	}
}

// ---------------------------------------------------------------------------
// resolveWidgetJobs unit tests
// ---------------------------------------------------------------------------

func TestResolveWidgetJobs_MetricRefWidgets(t *testing.T) {
	d := &dashboard.Dashboard{
		Name:       "test",
		Connection: "test-conn",
		Semantic: &dashboard.SemanticLayer{
			Source: &dashboard.Source{Table: "events"},
			Metrics: map[string]dashboard.Metric{
				"total":  {Aggregate: "count"},
				"amount": {Aggregate: "sum", Column: "amount"},
			},
		},
		Rows: []dashboard.Row{{
			Widgets: []dashboard.Widget{
				{Name: "Total", Type: "metric", MetricRef: "total"},
				{Name: "Amount", Type: "metric", MetricRef: "amount"},
			},
		}},
	}

	jobs, err := ResolveWidgetJobs(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Should produce 1 merged job instead of 2 individual jobs.
	if len(jobs) != 1 {
		t.Fatalf("expected 1 merged job, got %d", len(jobs))
	}
	if jobs[0].ID != MetricsJobID {
		t.Errorf("expected id %q, got %q", MetricsJobID, jobs[0].ID)
	}
	assertContains(t, jobs[0].SQL, "COUNT(*) AS total")
	assertContains(t, jobs[0].SQL, "SUM(amount) AS amount")
	assertEqual(t, jobs[0].Connection, "test-conn")
	// Fanout map should have both widgets.
	if len(jobs[0].MetricFanout) != 2 {
		t.Fatalf("expected 2 fanout entries, got %d", len(jobs[0].MetricFanout))
	}
	assertEqual(t, jobs[0].MetricFanout["r0-w0"], "total")
	assertEqual(t, jobs[0].MetricFanout["r0-w1"], "amount")
}

func TestResolveWidgetJobs_DimensionalWidgets(t *testing.T) {
	d := &dashboard.Dashboard{
		Name:       "test",
		Connection: "test-conn",
		Semantic: &dashboard.SemanticLayer{
			Source: &dashboard.Source{Table: "events"},
			Metrics: map[string]dashboard.Metric{
				"views": {Aggregate: "count", Filter: map[string]string{"event_name": "page_view"}},
			},
			Dimensions: map[string]dashboard.Dimension{
				"daily": {Column: "event_date", Type: "date"},
			},
		},
		Rows: []dashboard.Row{{
			Widgets: []dashboard.Widget{{
				Name:       "Traffic",
				Type:       "chart",
				Dimension:  "daily",
				MetricRefs: []string{"views"},
			}},
		}},
	}

	jobs, err := ResolveWidgetJobs(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	assertContains(t, jobs[0].SQL, "GROUP BY 1")
	assertContains(t, jobs[0].SQL, "event_date AS event_date")
	assertContains(t, jobs[0].SQL, "AS views")
}

func TestResolveWidgetJobs_RegularSQLWidgets(t *testing.T) {
	d := &dashboard.Dashboard{
		Name:       "test",
		Connection: "test-conn",
		Rows: []dashboard.Row{{
			Widgets: []dashboard.Widget{{
				Name: "Table",
				Type: "table",
				SQL:  "SELECT * FROM orders",
			}},
		}},
	}

	jobs, err := ResolveWidgetJobs(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	assertEqual(t, jobs[0].SQL, "SELECT * FROM orders")
}

func TestResolveWidgetJobs_SkipsTextDividerImage(t *testing.T) {
	d := &dashboard.Dashboard{
		Name: "test",
		Rows: []dashboard.Row{{
			Widgets: []dashboard.Widget{
				{Name: "txt", Type: "text", Content: "hi"},
				{Name: "div", Type: "divider"},
				{Name: "img", Type: "image", Src: "x.png"},
			},
		}},
	}

	jobs, err := ResolveWidgetJobs(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 0 {
		t.Fatalf("expected 0 jobs for text/divider/image, got %d", len(jobs))
	}
}

func TestResolveWidgetJobs_MixedWidgetTypes(t *testing.T) {
	d := &dashboard.Dashboard{
		Name:       "test",
		Connection: "conn",
		Semantic: &dashboard.SemanticLayer{
			Source: &dashboard.Source{Table: "events"},
			Metrics: map[string]dashboard.Metric{
				"total": {Aggregate: "count"},
			},
			Dimensions: map[string]dashboard.Dimension{
				"daily": {Column: "date", Type: "date"},
			},
		},
		Rows: []dashboard.Row{
			{Widgets: []dashboard.Widget{
				{Name: "Metric", Type: "metric", MetricRef: "total"},
				{Name: "Info", Type: "text", Content: "note"},
			}},
			{Widgets: []dashboard.Widget{
				{Name: "Chart", Type: "chart", Dimension: "daily", MetricRefs: []string{"total"}},
				{Name: "Table", Type: "table", SQL: "SELECT 1"},
			}},
		},
	}

	jobs, err := ResolveWidgetJobs(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Chart + Table + 1 merged metrics job = 3 (text is skipped).
	if len(jobs) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(jobs))
	}
	// Non-metric jobs come first, then the merged metrics job.
	assertEqual(t, jobs[0].ID, "r1-w0")      // chart
	assertEqual(t, jobs[1].ID, "r1-w1")      // table
	assertEqual(t, jobs[2].ID, MetricsJobID) // merged metrics
}

func TestResolveWidgetJobs_JinjaFiltersRendered(t *testing.T) {
	d := &dashboard.Dashboard{
		Name:       "test",
		Connection: "conn",
		Rows: []dashboard.Row{{
			Widgets: []dashboard.Widget{{
				Name: "Filtered",
				Type: "table",
				SQL:  "SELECT * FROM orders WHERE region = '{{ filters.region }}'",
			}},
		}},
	}

	filters := map[string]any{"region": "US"}
	jobs, err := ResolveWidgetJobs(d, filters)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatal("expected 1 job")
	}
	assertContains(t, jobs[0].SQL, "region = 'US'")
	assertNotContains(t, jobs[0].SQL, "{{")
}

func TestResolveWidgetJobs_DateFilterPassedToDeclarative(t *testing.T) {
	d := &dashboard.Dashboard{
		Name:       "test",
		Connection: "conn",
		Semantic: &dashboard.SemanticLayer{
			Source: &dashboard.Source{Table: "events", DateColumn: "event_date"},
			Metrics: map[string]dashboard.Metric{
				"total": {Aggregate: "count"},
			},
		},
		Filters: []dashboard.Filter{
			{Name: "date_range", Type: "date-range"},
		},
		Rows: []dashboard.Row{{
			Widgets: []dashboard.Widget{{
				Name:      "Count",
				Type:      "metric",
				MetricRef: "total",
			}},
		}},
	}

	filters := map[string]any{
		"date_range": map[string]any{"start": "2025-01-01", "end": "2025-12-31"},
	}
	jobs, err := ResolveWidgetJobs(d, filters)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatal("expected 1 merged job")
	}
	assertContains(t, jobs[0].SQL, "event_date >= '2025-01-01'")
	assertContains(t, jobs[0].SQL, "event_date <= '2025-12-31'")
}

func TestResolveWidgetJobs_UnknownDimensionError(t *testing.T) {
	d := &dashboard.Dashboard{
		Name:       "test",
		Connection: "conn",
		Semantic: &dashboard.SemanticLayer{
			Source:     &dashboard.Source{Table: "events"},
			Metrics:    map[string]dashboard.Metric{"total": {Aggregate: "count"}},
			Dimensions: map[string]dashboard.Dimension{},
		},
		Rows: []dashboard.Row{{
			Widgets: []dashboard.Widget{{
				Name:       "Bad",
				Type:       "chart",
				Dimension:  "nonexistent",
				MetricRefs: []string{"total"},
			}},
		}},
	}

	_, err := ResolveWidgetJobs(d, nil)
	if err == nil {
		t.Fatal("expected error for unknown dimension")
	}
	assertContains(t, err.Error(), "nonexistent")
}

func TestResolveWidgetJobs_SourceConnectionOverride(t *testing.T) {
	d := &dashboard.Dashboard{
		Name:       "test",
		Connection: "default-conn",
		Semantic: &dashboard.SemanticLayer{
			Source:  &dashboard.Source{Table: "events", Connection: "source-conn"},
			Metrics: map[string]dashboard.Metric{"total": {Aggregate: "count"}},
		},
		Rows: []dashboard.Row{{
			Widgets: []dashboard.Widget{{
				Name:      "Count",
				Type:      "metric",
				MetricRef: "total",
			}},
		}},
	}

	jobs, err := ResolveWidgetJobs(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	// The merged metrics job uses the source connection.
	assertEqual(t, jobs[0].Connection, "source-conn")
}

func TestResolveWidgetJobs_ExternalSemanticProjectDashboard(t *testing.T) {
	d, err := dashboard.LoadFile("../../testdata/project/dashboards/semantic-sales.yml")
	if err != nil {
		t.Fatal(err)
	}

	jobs, err := ResolveWidgetJobs(d, map[string]any{"country": "CA"})
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 6 {
		t.Fatalf("expected 6 jobs, got %d", len(jobs))
	}

	assertContains(t, jobs[0].SQL, "sum(amount) AS revenue")
	assertContains(t, jobs[1].SQL, "avg_order_value")
	assertContains(t, jobs[1].SQL, "country = 'CA'")
	assertContains(t, jobs[3].SQL, "date_trunc('month', order_date) AS order_date")
	assertContains(t, jobs[3].SQL, "ORDER BY order_date ASC")
	assertContains(t, jobs[4].SQL, "status = 'completed'")
	assertContains(t, jobs[4].SQL, "LIMIT 5")
	assertContains(t, jobs[5].SQL, "count(distinct order_id) AS order_count")
}

// ---------------------------------------------------------------------------
// Integration tests — batch endpoint with Google Analytics dashboard
// ---------------------------------------------------------------------------

func TestBatchQuery_GoogleAnalyticsDashboard(t *testing.T) {
	// Load the real GA dashboard to verify all widgets produce jobs.
	d, err := dashboard.LoadFile("../../testdata/dashboards/google-analytics.yml")
	if err != nil {
		t.Fatal(err)
	}

	// The mock needs to return columns that match the merged metrics query
	// (events, page_views, sessions, users — sorted alphabetically by GenerateMetricsSQL).
	mock := &mockBackend{
		result: &query.QueryResult{
			Columns: []query.ColumnInfo{
				{Name: "events"}, {Name: "page_views"}, {Name: "sessions"}, {Name: "users"},
			},
			Rows: [][]any{{5000.0, 1200.0, 400.0, 300.0}},
		},
	}
	s := &Server{
		backend: mock,
		loader:  &dashboardLoader{dir: "../../testdata/dashboards"},
	}
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("POST /api/v1/dashboards/{name}/data", s.handleBatchQuery)

	body := `{"filters":{"property":"Bruin Cloud","date_range":{"start":"2025-06-01","end":"2026-03-06"}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dashboards/Google%20Analytics/data", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	assertEqual(t, w.Code, http.StatusOK)

	var resp struct {
		Widgets map[string]*WidgetQueryResult `json:"widgets"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	// GA dashboard has 10 widgets total:
	// 4 metrics (row 0) + 2 dimensional charts (row 1) + 3 charts (row 2) + 1 table (row 3)
	expectedWidgets := []string{
		"r0-w0", "r0-w1", "r0-w2", "r0-w3", // metrics
		"r1-w0", "r1-w1", // dimensional charts
		"r2-w0", "r2-w1", "r2-w2", // charts (1 SQL, 2 dimensional)
		"r3-w0", // table
	}

	if len(resp.Widgets) != len(expectedWidgets) {
		t.Fatalf("expected %d widgets, got %d: %v", len(expectedWidgets), len(resp.Widgets), keys(resp.Widgets))
	}

	for _, id := range expectedWidgets {
		wr, ok := resp.Widgets[id]
		if !ok {
			t.Errorf("missing widget %q", id)
			continue
		}
		if wr.Error != "" {
			t.Errorf("widget %q has error: %s", id, wr.Error)
		}
		if wr.Query == "" {
			t.Errorf("widget %q has empty query", id)
		}
	}

	// With metric merging: 1 merged metrics + 2 dim charts (row 1) + 1 SQL chart + 2 dim charts (row 2) + 1 table = 7 backend calls.
	if len(mock.calls) != 7 {
		t.Errorf("expected 7 backend calls (4 metrics merged into 1), got %d", len(mock.calls))
	}

	// Check that Jinja was rendered (source table should have the resolved property).
	_ = d // used above for loading
	for _, call := range mock.calls {
		if strings.Contains(call.SQL, "{{") || strings.Contains(call.SQL, "{%") {
			t.Errorf("unrendered Jinja template in SQL: %s", call.SQL[:min(80, len(call.SQL))])
		}
	}
}

func TestStreamQuery_GoogleAnalyticsDashboard(t *testing.T) {
	mock := &mockBackend{
		result: &query.QueryResult{
			Columns: []query.ColumnInfo{
				{Name: "events"}, {Name: "page_views"}, {Name: "sessions"}, {Name: "users"},
			},
			Rows: [][]any{{5000.0, 1200.0, 400.0, 300.0}},
		},
	}
	s := &Server{
		backend: mock,
		loader:  &dashboardLoader{dir: "../../testdata/dashboards"},
	}
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("POST /api/v1/dashboards/{name}/stream", s.handleStreamQuery)

	body := `{"filters":{"property":"Bruin Cloud","date_range":{"start":"2025-06-01","end":"2026-03-06"}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dashboards/Google%20Analytics/stream", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	assertEqual(t, w.Code, http.StatusOK)

	// Content type should be NDJSON.
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "ndjson") {
		t.Errorf("expected ndjson content type, got %q", ct)
	}

	// Parse all NDJSON lines.
	lines := strings.Split(strings.TrimSpace(w.Body.String()), "\n")
	if len(lines) != 10 {
		t.Fatalf("expected 10 NDJSON lines, got %d", len(lines))
	}

	widgetIDs := make(map[string]bool)
	for _, line := range lines {
		var msg struct {
			ID   string             `json:"id"`
			Data *WidgetQueryResult `json:"data"`
		}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			t.Fatalf("failed to parse NDJSON line: %v\nline: %s", err, line)
		}
		widgetIDs[msg.ID] = true
		if msg.Data == nil {
			t.Errorf("widget %q has nil data", msg.ID)
		}
		if msg.Data != nil && msg.Data.Query == "" {
			t.Errorf("widget %q has empty query", msg.ID)
		}
	}

	// Should have all 10 widget IDs.
	if len(widgetIDs) != 10 {
		t.Errorf("expected 10 unique widget IDs, got %d", len(widgetIDs))
	}
}

func TestBatchQuery_RegularDashboard(t *testing.T) {
	mock := &mockBackend{
		result: &query.QueryResult{
			Columns: []query.ColumnInfo{{Name: "total"}},
			Rows:    [][]any{{100}},
		},
	}
	s := &Server{
		backend: mock,
		loader:  &dashboardLoader{dir: "../../testdata/dashboards"},
	}
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("POST /api/v1/dashboards/{name}/data", s.handleBatchQuery)

	body := `{"filters":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dashboards/Sales%20Analytics/data", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	assertEqual(t, w.Code, http.StatusOK)

	var resp struct {
		Widgets map[string]*WidgetQueryResult `json:"widgets"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	// Sales dashboard has 11 widgets, all SQL-based.
	if len(resp.Widgets) == 0 {
		t.Fatal("expected widgets in response")
	}

	for id, wr := range resp.Widgets {
		if wr.Error != "" {
			t.Errorf("widget %q has error: %s", id, wr.Error)
		}
	}
}

func TestBatchQuery_ProjectSemanticDashboard(t *testing.T) {
	mock := &mockBackend{
		result: &query.QueryResult{
			Columns: []query.ColumnInfo{{Name: "value"}},
			Rows:    [][]any{{100}},
		},
	}
	s := &Server{
		backend: mock,
		paths:   dashboard.ResolveProjectPaths("../../testdata/project"),
		loader:  &dashboardLoader{dir: "../../testdata/project"},
	}
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("POST /api/v1/dashboards/{name}/data", s.handleBatchQuery)

	body := `{"filters":{"country":"CA"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dashboards/Semantic%20Sales/data", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	assertEqual(t, w.Code, http.StatusOK)

	var resp struct {
		Widgets map[string]*WidgetQueryResult `json:"widgets"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if len(resp.Widgets) != 6 {
		t.Fatalf("expected 6 widgets, got %d", len(resp.Widgets))
	}
	if len(mock.calls) != 6 {
		t.Fatalf("expected 6 backend calls, got %d", len(mock.calls))
	}
	for _, call := range mock.calls {
		assertNotContains(t, call.SQL, "{{")
		assertNotContains(t, call.SQL, "{%")
	}
}

func TestBatchQuery_NotFound(t *testing.T) {
	s := testServer(t)
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dashboards/nonexistent/data", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	assertEqual(t, w.Code, http.StatusNotFound)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("missing %q in:\n  %s", substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("should not contain %q in:\n  %s", substr, s)
	}
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
