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

	var jobs []widgetJob

	for i, row := range d.Rows {
		for j, widget := range row.Widgets {
			if widget.Type == "text" {
				continue
			}
			sql, conn, err := widget.ResolvedQuery(d)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			// Template filter values into the query.
			if req.Filters != nil {
				sql, err = tmpl.Render(sql, req.Filters)
				if err != nil {
					writeError(w, http.StatusBadRequest, "template error: "+err.Error())
					return
				}
			}

			id := widgetID(i, j)
			jobs = append(jobs, widgetJob{id: id, sql: sql, connection: conn})
		}
	}

	// Group jobs by connection so same-connection queries run sequentially
	// (avoids file lock contention for DuckDB, connection pool exhaustion, etc.)
	// Different connections run in parallel.
	connGroups := make(map[string][]widgetJob)
	for _, job := range jobs {
		connGroups[job.connection] = append(connGroups[job.connection], job)
	}

	results := make(map[string]*widgetQueryResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, group := range connGroups {
		wg.Add(1)
		go func(groupJobs []widgetJob) {
			defer wg.Done()
			for _, j := range groupJobs {
				wr := executeWidgetQuery(r.Context(), s.backend, j)
				mu.Lock()
				results[j.id] = wr
				mu.Unlock()
			}
		}(group)
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
