package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/bruin-data/dac/pkg/dashboard"
	"github.com/bruin-data/dac/pkg/query"
	tmpl "github.com/bruin-data/dac/pkg/template"
)

type widgetJob struct {
	id         string
	sql        string
	connection string
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("error encoding JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

type dashboardSummary struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Connection  string `json:"connection,omitempty"`
	WidgetCount int    `json:"widget_count"`
	FilterCount int    `json:"filter_count"`
	RowCount    int    `json:"row_count"`
}

func (s *Server) handleListDashboards(w http.ResponseWriter, r *http.Request) {
	dashboards, err := s.loader.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	summaries := make([]dashboardSummary, 0, len(dashboards))
	for _, d := range dashboards {
		widgetCount := 0
		for _, row := range d.Rows {
			widgetCount += len(row.Widgets)
		}
		summaries = append(summaries, dashboardSummary{
			Name:        d.Name,
			Description: d.Description,
			Connection:  d.Connection,
			WidgetCount: widgetCount,
			FilterCount: len(d.Filters),
			RowCount:    len(d.Rows),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"dashboards": summaries})
}

func (s *Server) handleGetDashboard(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	dashboards, err := s.loader.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, d := range dashboards {
		if d.Name == name {
			writeJSON(w, http.StatusOK, d)
			return
		}
	}
	writeError(w, http.StatusNotFound, "dashboard not found: "+name)
}

func (s *Server) handleGetDashboardRaw(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	dashboards, err := s.loader.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, d := range dashboards {
		if d.Name == name {
			data, err := os.ReadFile(d.FilePath)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to read dashboard file: "+err.Error())
				return
			}
			w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
			w.Write(data)
			return
		}
	}
	writeError(w, http.StatusNotFound, "dashboard not found: "+name)
}

type batchQueryRequest struct {
	Filters map[string]any `json:"filters"`
}

type widgetQueryResult struct {
	Columns []struct {
		Name string `json:"name"`
		Type string `json:"type,omitempty"`
	} `json:"columns"`
	Rows  [][]any `json:"rows"`
	Query string  `json:"query,omitempty"`
	Error string  `json:"error,omitempty"`
}

func (s *Server) handleBatchQuery(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var req batchQueryRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}
	}

	// Find the dashboard.
	dashboards, err := s.loader.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var d *dashboard.Dashboard
	for _, dash := range dashboards {
		if dash.Name == name {
			d = dash
			break
		}
	}
	if d == nil {
		writeError(w, http.StatusNotFound, "dashboard not found: "+name)
		return
	}

	// Merge request filters over dashboard defaults so unset filters
	// still have values for query templating.
	filters := d.DefaultFilters()
	for k, v := range req.Filters {
		filters[k] = v
	}

	jobs, err := s.resolveWidgetJobs(d, filters)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Execute all widget queries concurrently with a concurrency limit.
	results := make(map[string]*widgetQueryResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	sem := make(chan struct{}, 8)
	for _, j := range jobs {
		wg.Add(1)
		go func(j widgetJob) {
			defer wg.Done()
			sem <- struct{}{}
			wr := executeWidgetQuery(r.Context(), s.backend, j)
			<-sem
			mu.Lock()
			results[j.id] = wr
			mu.Unlock()
		}(j)
	}

	wg.Wait()
	writeJSON(w, http.StatusOK, map[string]any{"widgets": results})
}

func (s *Server) handleSingleQuery(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Connection string `json:"connection"`
		SQL        string `json:"sql"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	result, err := s.backend.Execute(r.Context(), req.Connection, req.SQL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	templateName := s.config.TemplateName
	resp := map[string]any{
		"template":      templateName,
		"admin_enabled": s.config.AdminPassword != "",
	}

	// If the template is a user-defined theme (not a built-in), include its tokens
	// so the frontend can apply them even without having the template's components.
	if t, ok := s.themes.Get(templateName); ok {
		resp["tokens"] = t.Tokens
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleListThemes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"themes": s.themes.List()})
}

func (s *Server) handleGetTheme(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	t, ok := s.themes.Get(name)
	if !ok {
		writeError(w, http.StatusNotFound, "theme not found: "+name)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// resolveWidgetJobs builds the list of SQL jobs for all data-bearing widgets
// in a dashboard, including declarative metric and dimensional widgets.
func (s *Server) resolveWidgetJobs(d *dashboard.Dashboard, filters map[string]any) ([]widgetJob, error) {
	// Extract date filter values for declarative SQL generation.
	var dateFilter map[string]any
	if dfName := d.DateRangeFilterName(); dfName != "" {
		if df, ok := filters[dfName]; ok {
			if m, ok := df.(map[string]any); ok {
				dateFilter = m
			}
		}
	}

	var jobs []widgetJob
	for i, row := range d.Rows {
		for j, widget := range row.Widgets {
			if widget.Type == dashboard.WidgetTypeText || widget.Type == dashboard.WidgetTypeDivider || widget.Type == dashboard.WidgetTypeImage {
				continue
			}

			var sql, conn string
			var err error

			switch {
			case widget.MetricRef != "" && d.Source != nil:
				// Declarative metric widget: generate scalar query.
				sql, err = generateSingleMetricSQL(d, widget.MetricRef, dateFilter)
				if err != nil {
					return nil, fmt.Errorf("widget %q: %w", widget.Name, err)
				}
				conn = d.SourceConnection()

			case len(widget.MetricRefs) > 0 && widget.Dimension != "" && d.Source != nil:
				// Declarative dimensional chart widget.
				dim, ok := d.Dimensions[widget.Dimension]
				if !ok {
					return nil, fmt.Errorf("widget %q: dimension %q not found", widget.Name, widget.Dimension)
				}
				sql, err = dashboard.GenerateDimensionalSQL(d.Source, d.Metrics, widget.MetricRefs, &dim, dateFilter, widget.Limit)
				if err != nil {
					return nil, fmt.Errorf("widget %q: %w", widget.Name, err)
				}
				conn = d.SourceConnection()

			default:
				sql, conn, err = widget.ResolvedQuery(d)
				if err != nil {
					return nil, err
				}
				if sql == "" {
					continue
				}
			}

			// Render Jinja templates (e.g. source table name with conditionals).
			if len(filters) > 0 {
				sql, err = tmpl.Render(sql, filters)
				if err != nil {
					return nil, fmt.Errorf("template error: %w", err)
				}
			}
			jobs = append(jobs, widgetJob{id: widgetID(i, j), sql: sql, connection: conn})
		}
	}
	return jobs, nil
}

// generateSingleMetricSQL builds a scalar SQL query for a single metric widget.
// Expression metrics are inlined as SQL so the database evaluates them directly.
func generateSingleMetricSQL(d *dashboard.Dashboard, metricName string, dateFilter map[string]any) (string, error) {
	m, ok := d.Metrics[metricName]
	if !ok {
		return "", fmt.Errorf("metric %q not found", metricName)
	}

	// For expression metrics, use GenerateDimensionalSQL machinery but without a GROUP BY.
	// Build a simple SELECT <expr> as value FROM <source> WHERE <date>.
	if m.IsExpression() {
		// Generate the merged metrics query and wrap the expression.
		metricsSQL, err := dashboard.GenerateMetricsSQL(d.Source, d.Metrics, dateFilter)
		if err != nil {
			return "", err
		}
		// The merged query returns all aggregate metrics as columns. Wrap it
		// in a subquery and compute the expression.
		return fmt.Sprintf("SELECT (%s) as value FROM (%s)", m.Expression, metricsSQL), nil
	}

	// Direct aggregate metric: generate a single-column query.
	onlyThis := map[string]dashboard.Metric{metricName: m}
	return dashboard.GenerateMetricsSQL(d.Source, onlyThis, dateFilter)
}

func executeWidgetQuery(ctx context.Context, backend query.Backend, j widgetJob) *widgetQueryResult {
	qr, err := backend.Execute(ctx, j.connection, j.sql)
	if err != nil {
		return &widgetQueryResult{Query: j.sql, Error: err.Error()}
	}

	wr := &widgetQueryResult{
		Rows:  make([][]any, len(qr.Rows)),
		Query: j.sql,
	}
	for _, col := range qr.Columns {
		wr.Columns = append(wr.Columns, struct {
			Name string `json:"name"`
			Type string `json:"type,omitempty"`
		}{Name: col.Name, Type: col.Type})
	}
	for i, row := range qr.Rows {
		wr.Rows[i] = row
	}
	return wr
}

// handleStreamQuery is the streaming variant of handleBatchQuery.
// It writes each widget result as a newline-delimited JSON line
// as soon as the query completes, so the frontend can render
// widgets incrementally.
func (s *Server) handleStreamQuery(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	name := r.PathValue("name")

	var req batchQueryRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}
	}

	dashboards, err := s.loader.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var d *dashboard.Dashboard
	for _, dash := range dashboards {
		if dash.Name == name {
			d = dash
			break
		}
	}
	if d == nil {
		writeError(w, http.StatusNotFound, "dashboard not found: "+name)
		return
	}

	filters := d.DefaultFilters()
	for k, v := range req.Filters {
		filters[k] = v
	}

	jobs, err := s.resolveWidgetJobs(d, filters)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Set up streaming response.
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Results channel — each goroutine sends its result here.
	type streamResult struct {
		ID   string             `json:"id"`
		Data *widgetQueryResult `json:"data"`
	}
	ch := make(chan streamResult, len(jobs))

	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Add(1)
		go func(j widgetJob) {
			defer wg.Done()
			sem <- struct{}{}
			wr := executeWidgetQuery(r.Context(), s.backend, j)
			<-sem
			ch <- streamResult{ID: j.id, Data: wr}
		}(j)
	}

	// Close channel when all goroutines complete.
	go func() {
		wg.Wait()
		close(ch)
	}()

	enc := json.NewEncoder(w)
	for result := range ch {
		if err := enc.Encode(result); err != nil {
			return // client disconnected
		}
		flusher.Flush()
	}
}

func widgetID(rowIdx, widgetIdx int) string {
	return fmt.Sprintf("r%d-w%d", rowIdx, widgetIdx)
}
