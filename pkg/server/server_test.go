package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	s, err := New(Config{
		DashboardDir: "../../testdata/dashboards",
		TemplateName: "bruin",
		Port:         0,
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func assertEqual(t *testing.T, got, want any) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestListDashboards(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboards", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	assertEqual(t, w.Code, http.StatusOK)

	var resp struct {
		Dashboards []json.RawMessage `json:"dashboards"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, len(resp.Dashboards), 3)
}

func TestGetDashboard(t *testing.T) {
	s := testServer(t)

	t.Run("existing dashboard", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboards/Sales%20Analytics", nil)
		w := httptest.NewRecorder()
		s.mux.ServeHTTP(w, req)

		assertEqual(t, w.Code, http.StatusOK)

		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
		assertEqual(t, resp["name"], "Sales Analytics")
	})

	t.Run("nonexistent dashboard", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboards/nonexistent", nil)
		w := httptest.NewRecorder()
		s.mux.ServeHTTP(w, req)

		assertEqual(t, w.Code, http.StatusNotFound)
	})
}

func TestGetDashboardRaw(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboards/Sales%20Analytics/raw", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	assertEqual(t, w.Code, http.StatusOK)

	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/yaml") {
		t.Errorf("content-type = %q, want text/yaml", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, "name: Sales Analytics") {
		t.Errorf("body does not contain expected YAML name field")
	}
}

func TestConfig(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	assertEqual(t, w.Code, http.StatusOK)

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if _, ok := resp["template"]; !ok {
		t.Error("response missing 'template' field")
	}
	assertEqual(t, resp["template"], "bruin")
}

func TestListThemes(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/themes", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	assertEqual(t, w.Code, http.StatusOK)

	var resp struct {
		Themes []any `json:"themes"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Themes) == 0 {
		t.Error("expected at least one theme")
	}
}
