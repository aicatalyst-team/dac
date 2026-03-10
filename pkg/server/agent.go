package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/bruin-data/dac/pkg/codex"
	"github.com/bruin-data/dac/pkg/dashboard"
)

// Shared dashboard schema reference used by both prompts.
const dashboardSchemaRef = `Dashboard YAML structure:
- name, description, connection (default DB connection)
- filters: [{name, type (select|date-range), default, options}]
- queries: named reusable SQL queries (sql: inline | file: path.sql)
- rows: [{widgets: [...]}]

Widget types:
- metric: KPI card. Fields: query/sql/file, column, prefix, suffix, format (number|currency|percent), col
- chart: Visualization. Fields: chart (line|bar|area|pie|scatter|funnel|histogram), x, y:[], label, value, stacked, col
- table: Data table. Fields: sql/query/file, columns: [{name, label, format}], col
- text: Markdown block. Fields: content, col
- divider: Horizontal separator. col
- image: Static image. Fields: src, alt, col

Grid: 12-column. Each widget has col:N (1-12). Widgets in a row should sum to 12.

Query templating (Jinja):
- {{ filters.date_range.start }}, {{ filters.date_range.end }}
- {% if filters.region != 'All' %} AND region = '{{ filters.region }}' {% endif %}
`

// System prompt prepended to the first turn of each thread.
const agentSystemPrompt = `You are a dashboard editor for "dac" (Dashboard-as-Code). The user is viewing a dashboard in their browser and wants you to modify it. After you edit a YAML file, the dashboard reloads automatically.

` + dashboardSchemaRef + `
Rules:
- ALWAYS read the dashboard file before modifying it
- Make minimal, targeted edits — don't rewrite the whole file
- Preserve existing formatting and comments
- When adding widgets, respect the 12-column grid
`

// System prompt for creating a new dashboard from scratch.
const agentCreatePrompt = `You are a dashboard builder for "dac" (Dashboard-as-Code). The user wants to create a new dashboard. Help them build one by writing a YAML file. After you create the file, the dashboard appears automatically.

` + dashboardSchemaRef + `
Rules:
- Create a new .yml file in the dashboard directory (given below)
- Pick a short, descriptive filename like "sales.yml" or "traffic-overview.yml"
- Use the user's description to decide which widgets, charts, and metrics to include
- Ask the user for the database connection name if they don't mention one
- Write complete, working SQL queries — prefer simple aggregations
- Start with 2-4 KPI metrics at the top, then charts, then a detail table
- Respond concisely — create the file, don't over-explain
`

type createSessionRequest struct {
	Dashboard string `json:"dashboard"`
	Model     string `json:"model,omitempty"`
}

type createSessionResponse struct {
	SessionID string `json:"session_id"`
}

func (s *Server) handleAgentCreateSession(w http.ResponseWriter, r *http.Request) {
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	threadID, err := s.codex.StartThread(s.config.DashboardDir, req.Model)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start agent session: "+err.Error())
		return
	}

	// Remember which dashboard this session is for.
	s.sessionDashMu.Lock()
	s.sessionDash[threadID] = req.Dashboard
	s.sessionDashMu.Unlock()

	writeJSON(w, http.StatusOK, createSessionResponse{SessionID: threadID})
}

type sendMessageRequest struct {
	Message string `json:"message"`
}

func (s *Server) handleAgentSendMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if strings.TrimSpace(req.Message) == "" {
		writeError(w, http.StatusBadRequest, "message cannot be empty")
		return
	}

	var input []map[string]any

	// On the first turn, prepend system context with the specific dashboard YAML.
	if !s.codex.IsThreadInitialized(sessionID) {
		s.sessionDashMu.RLock()
		dashName := s.sessionDash[sessionID]
		s.sessionDashMu.RUnlock()

		var prompt string
		if dashName == "__create__" {
			prompt = agentCreatePrompt
		} else {
			prompt = agentSystemPrompt
		}
		context := prompt + s.buildDashboardContext() + s.buildActiveDashboardContext(dashName)
		input = append(input, map[string]any{"type": "text", "text": context})
		s.codex.MarkThreadInitialized(sessionID)
		log.Printf("agent: first turn for thread %s, dashboard %q (%d bytes context)", sessionID, dashName, len(context))
	}

	input = append(input, map[string]any{"type": "text", "text": req.Message})

	// Log the full payload for debugging.
	payload, _ := json.Marshal(input)
	log.Printf("agent: turn payload (%d bytes): %s", len(payload), string(payload)[:min(500, len(payload))])

	if err := s.codex.StartTurn(sessionID, input, s.config.AgentEffort); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to send message: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// buildDashboardContext produces a summary of available dashboard files.
func (s *Server) buildDashboardContext() string {
	dashboards, err := s.loader.Load()
	if err != nil {
		return fmt.Sprintf("\nDashboard directory: %s\n", s.config.DashboardDir)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\nDashboard directory: %s\n", s.config.DashboardDir))
	b.WriteString("Available dashboards:\n")
	for _, d := range dashboards {
		fp := d.FilePath
		if fp == "" {
			fp = filepath.Join(s.config.DashboardDir, d.Name+".yml")
		}
		b.WriteString(fmt.Sprintf("- %s → %s", d.Name, fp))
		if d.Description != "" {
			b.WriteString(fmt.Sprintf(" (%s)", d.Description))
		}
		b.WriteByte('\n')
		b.WriteString(fmt.Sprintf("  widgets: %d, filters: %d, connection: %s\n",
			countWidgets(d), len(d.Filters), d.Connection))
	}
	return b.String()
}

// buildActiveDashboardContext tells the agent which dashboard the user is viewing and where to find it.
func (s *Server) buildActiveDashboardContext(dashName string) string {
	if dashName == "" {
		return ""
	}

	dashboards, err := s.loader.Load()
	if err != nil {
		return ""
	}

	var filePath string
	for _, d := range dashboards {
		if d.Name == dashName {
			filePath = d.FilePath
			break
		}
	}
	if filePath == "" {
		return ""
	}

	return fmt.Sprintf("\nThe user is currently viewing the \"%s\" dashboard. Its definition is at: %s\nAlways read this file before making changes.\n", dashName, filePath)
}

func countWidgets(d *dashboard.Dashboard) int {
	n := 0
	for _, row := range d.Rows {
		n += len(row.Widgets)
	}
	return n
}

// handleAgentEvents streams codex events to the frontend via SSE.
func (s *Server) handleAgentEvents(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := s.codex.Subscribe(sessionID)
	defer s.codex.Unsubscribe(sessionID, ch)

	// Send initial connected event.
	fmt.Fprintf(w, "data: %s\n\n", mustJSON(map[string]string{"type": "connected"}))
	flusher.Flush()

	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return
			}
			sseEvent := translateEvent(event)
			if sseEvent == nil {
				continue
			}
			data, _ := json.Marshal(sseEvent)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleAgentInterrupt(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")

	var req struct {
		TurnID string `json:"turn_id"`
	}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
	}

	if err := s.codex.InterruptTurn(sessionID, req.TurnID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to interrupt: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "interrupted"})
}

// translateEvent converts a raw codex notification into a frontend-friendly SSE payload.
func translateEvent(event *codex.Event) map[string]any {
	switch event.Method {
	case "item/agentMessage/delta":
		var p struct {
			Delta string `json:"delta"`
		}
		json.Unmarshal(event.Params, &p) //nolint:errcheck
		return map[string]any{"type": "agent_delta", "text": p.Delta}

	case "item/reasoning/summaryTextDelta":
		var p struct {
			Delta string `json:"delta"`
		}
		json.Unmarshal(event.Params, &p) //nolint:errcheck
		return map[string]any{"type": "reasoning_delta", "text": p.Delta}

	case "item/started":
		var p struct {
			Item json.RawMessage `json:"item"`
		}
		json.Unmarshal(event.Params, &p) //nolint:errcheck
		item := parseItem(p.Item)
		if item == nil {
			return nil
		}
		return map[string]any{"type": "item_started", "item": item}

	case "item/completed":
		var p struct {
			Item json.RawMessage `json:"item"`
		}
		json.Unmarshal(event.Params, &p) //nolint:errcheck
		item := parseItem(p.Item)
		if item == nil {
			return nil
		}
		return map[string]any{"type": "item_completed", "item": item}

	case "item/commandExecution/outputDelta":
		var p struct {
			Delta string `json:"delta"`
		}
		json.Unmarshal(event.Params, &p) //nolint:errcheck
		return map[string]any{"type": "command_output_delta", "output": p.Delta}

	case "turn/started":
		var p struct {
			Turn struct {
				ID string `json:"id"`
			} `json:"turn"`
		}
		json.Unmarshal(event.Params, &p) //nolint:errcheck
		return map[string]any{"type": "turn_started", "turn_id": p.Turn.ID}

	case "turn/completed":
		var p struct {
			Turn struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"turn"`
		}
		json.Unmarshal(event.Params, &p) //nolint:errcheck
		return map[string]any{"type": "turn_completed", "turn_id": p.Turn.ID, "status": p.Turn.Status}

	default:
		return nil
	}
}

// parseItem extracts type-specific fields from a codex item.
func parseItem(raw json.RawMessage) map[string]any {
	if raw == nil {
		return nil
	}

	var base struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	json.Unmarshal(raw, &base) //nolint:errcheck

	switch base.Type {
	case "agentMessage":
		var item struct {
			ID    string `json:"id"`
			Text  string `json:"text"`
			Phase string `json:"phase"`
		}
		json.Unmarshal(raw, &item) //nolint:errcheck
		m := map[string]any{"id": item.ID, "kind": "agentMessage", "text": item.Text}
		if item.Phase != "" {
			m["phase"] = item.Phase
		}
		return m

	case "commandExecution":
		var item struct {
			ID               string `json:"id"`
			Command          string `json:"command"`
			Cwd              string `json:"cwd"`
			Status           string `json:"status"`
			ExitCode         *int   `json:"exitCode"`
			AggregatedOutput string `json:"aggregatedOutput"`
		}
		json.Unmarshal(raw, &item) //nolint:errcheck
		return map[string]any{
			"id": item.ID, "kind": "commandExecution",
			"command": item.Command, "cwd": item.Cwd, "status": item.Status,
			"exitCode": item.ExitCode, "output": item.AggregatedOutput,
		}

	case "fileChange":
		var item struct {
			ID      string `json:"id"`
			Status  string `json:"status"`
			Changes []struct {
				FilePath string `json:"filePath"`
			} `json:"changes"`
		}
		json.Unmarshal(raw, &item) //nolint:errcheck
		files := make([]string, 0, len(item.Changes))
		for _, c := range item.Changes {
			files = append(files, c.FilePath)
		}
		return map[string]any{"id": item.ID, "kind": "fileChange", "status": item.Status, "files": files}

	case "reasoning":
		var item struct {
			ID      string `json:"id"`
			Summary string `json:"summary"`
		}
		json.Unmarshal(raw, &item) //nolint:errcheck
		return map[string]any{"id": item.ID, "kind": "reasoning", "text": item.Summary}

	default:
		return nil
	}
}

func mustJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("mustJSON error: %v", err)
		return `{}`
	}
	return string(data)
}
