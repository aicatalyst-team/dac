package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
}

func (s *Server) handleListDashboards(w http.ResponseWriter, r *http.Request) {
	dashboards, err := s.loader.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	summaries := make([]dashboardSummary, 0, len(dashboards))
	for _, d := range dashboards {
		summaries = append(summaries, dashboardSummary{
			Name:        d.Name,
			Description: d.Description,
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

type batchQueryRequest struct {
	Filters map[string]any `json:"filters"`
}

type widgetQueryResult struct {
	Columns []struct {
		Name string `json:"name"`
		Type string `json:"type,omitempty"`
	} `json:"columns"`
	Rows  [][]any `json:"rows"`
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

	var jobs []widgetJob

	// Track metric-ref widgets separately — they share a single merged query.
	type metricRefWidget struct {
		id        string
		metricRef string
	}
	var metricRefWidgets []metricRefWidget

	for i, row := range d.Rows {
		for j, widget := range row.Widgets {
			if widget.Type == dashboard.WidgetTypeText || widget.Type == dashboard.WidgetTypeDivider || widget.Type == dashboard.WidgetTypeImage {
				continue
			}

			id := widgetID(i, j)

			// Metric-ref widgets are handled via the merged metrics query.
			if widget.MetricRef != "" {
				metricRefWidgets = append(metricRefWidgets, metricRefWidget{id: id, metricRef: widget.MetricRef})
				continue
			}

			// Dimensional chart widgets generate SQL from source + metrics.
			if len(widget.MetricRefs) > 0 && widget.Dimension != "" && d.Source != nil {
				dimSQL, dimErr := s.buildDimensionalQuery(d, &widget, filters)
				if dimErr != nil {
					writeError(w, http.StatusBadRequest, dimErr.Error())
					return
				}
				jobs = append(jobs, widgetJob{id: id, sql: dimSQL, connection: d.SourceConnection()})
				continue
			}

			sql, conn, err := widget.ResolvedQuery(d)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			// Template filter values into the query.
			if len(filters) > 0 {
				sql, err = tmpl.Render(sql, filters)
				if err != nil {
					writeError(w, http.StatusBadRequest, "template error: "+err.Error())
					return
				}
			}

			jobs = append(jobs, widgetJob{id: id, sql: sql, connection: conn})
		}
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

	// If there are metric-ref widgets, generate and execute the merged metrics query.
	if len(metricRefWidgets) > 0 && d.Source != nil && len(d.Metrics) > 0 {
		metricsResults, err := s.executeMetrics(r.Context(), d, filters)
		if err != nil {
			// Store error on each metric widget rather than failing the whole request.
			for _, mw := range metricRefWidgets {
				results[mw.id] = &widgetQueryResult{Error: err.Error()}
			}
		} else {
			for _, mw := range metricRefWidgets {
				val, ok := metricsResults[mw.metricRef]
				if !ok {
					results[mw.id] = &widgetQueryResult{Error: fmt.Sprintf("metric %q not found", mw.metricRef)}
					continue
				}
				results[mw.id] = &widgetQueryResult{
					Columns: []struct {
						Name string `json:"name"`
						Type string `json:"type,omitempty"`
					}{{Name: "value", Type: "FLOAT"}},
					Rows: [][]any{{val}},
				}
			}
		}
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
	resp := map[string]any{"template": templateName}

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

// buildDimensionalQuery generates SQL for a chart widget that uses
// dimension + metrics instead of raw SQL.
func (s *Server) buildDimensionalQuery(d *dashboard.Dashboard, widget *dashboard.Widget, filters map[string]any) (string, error) {
	dim, ok := d.Dimensions[widget.Dimension]
	if !ok {
		return "", fmt.Errorf("widget %q: dimension %q not found in dimensions map", widget.Name, widget.Dimension)
	}

	var dateFilter map[string]any
	if drName := d.DateRangeFilterName(); drName != "" {
		if v, ok := filters[drName]; ok {
			if m, ok := v.(map[string]any); ok {
				dateFilter = m
			}
		}
	}

	sql, err := dashboard.GenerateDimensionalSQL(d.Source, d.Metrics, widget.MetricRefs, &dim, dateFilter, widget.Limit)
	if err != nil {
		return "", err
	}

	if len(filters) > 0 {
		sql, err = tmpl.Render(sql, filters)
		if err != nil {
			return "", fmt.Errorf("template error in dimensional query: %w", err)
		}
	}

	return sql, nil
}

// executeMetrics generates a merged SQL query for all aggregate metrics,
// executes it once, and then evaluates expression metrics from the results.
func (s *Server) executeMetrics(ctx context.Context, d *dashboard.Dashboard, filters map[string]any) (map[string]float64, error) {
	// Find the date-range filter value for automatic date filtering.
	var dateFilter map[string]any
	if drName := d.DateRangeFilterName(); drName != "" {
		if v, ok := filters[drName]; ok {
			if m, ok := v.(map[string]any); ok {
				dateFilter = m
			}
		}
	}

	sql, err := dashboard.GenerateMetricsSQL(d.Source, d.Metrics, dateFilter)
	if err != nil {
		return nil, err
	}

	// Template Jinja expressions (e.g. in the source table name).
	if len(filters) > 0 {
		sql, err = tmpl.Render(sql, filters)
		if err != nil {
			return nil, fmt.Errorf("template error in metrics query: %w", err)
		}
	}

	conn := d.SourceConnection()
	qr, err := s.backend.Execute(ctx, conn, sql)
	if err != nil {
		return nil, err
	}

	// Extract values from the single result row.
	values := make(map[string]float64)
	if len(qr.Rows) > 0 {
		for i, col := range qr.Columns {
			if i < len(qr.Rows[0]) {
				values[col.Name] = toFloat64(qr.Rows[0][i])
			}
		}
	}

	// Evaluate expression metrics.
	for _, name := range dashboard.ExpressionMetrics(d.Metrics) {
		m := d.Metrics[name]
		val, err := dashboard.EvaluateExpression(m.Expression, values)
		if err != nil {
			return nil, fmt.Errorf("metric %q expression error: %w", name, err)
		}
		values[name] = val
	}

	return values, nil
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	case string:
		var f float64
		fmt.Sscanf(n, "%f", &f)
		return f
	default:
		return 0
	}
}

func executeWidgetQuery(ctx context.Context, backend query.Backend, j widgetJob) *widgetQueryResult {
	qr, err := backend.Execute(ctx, j.connection, j.sql)
	if err != nil {
		return &widgetQueryResult{Error: err.Error()}
	}

	wr := &widgetQueryResult{
		Rows: make([][]any, len(qr.Rows)),
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

func widgetID(rowIdx, widgetIdx int) string {
	return fmt.Sprintf("r%d-w%d", rowIdx, widgetIdx)
}
