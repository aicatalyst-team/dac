package dashboard

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Loader tests
// ---------------------------------------------------------------------------

func TestLoadDir_LoadsAllDashboards(t *testing.T) {
	dashboards, err := LoadDir("../../testdata/dashboards")
	assertNoErr(t, err)

	// 3 YAML + 2 TSX = 5
	if len(dashboards) != 5 {
		t.Fatalf("expected 5 dashboards, got %d", len(dashboards))
	}
}

func TestLoadFile_SalesYAML(t *testing.T) {
	d, err := LoadFile("../../testdata/dashboards/sales.yml")
	assertNoErr(t, err)

	if d.Name != "Sales Analytics" {
		t.Errorf("expected name %q, got %q", "Sales Analytics", d.Name)
	}
	if d.Connection != "local_duckdb" {
		t.Errorf("expected connection %q, got %q", "local_duckdb", d.Connection)
	}
	if len(d.Filters) != 2 {
		t.Errorf("expected 2 filters, got %d", len(d.Filters))
	}
	if len(d.Rows) != 5 {
		t.Errorf("expected 5 rows, got %d", len(d.Rows))
	}
}

func TestLoadFile_ResolvesExternalSQLFiles(t *testing.T) {
	d, err := LoadFile("../../testdata/dashboards/sales.yml")
	assertNoErr(t, err)

	q, ok := d.Queries["revenue_by_month"]
	if !ok {
		t.Fatal("expected query 'revenue_by_month' to exist")
	}
	if q.SQL == "" {
		t.Fatal("expected revenue_by_month SQL to be resolved from file, got empty string")
	}
	if !strings.Contains(q.SQL, "DATE_TRUNC") {
		t.Errorf("expected resolved SQL to contain DATE_TRUNC, got:\n  %s", q.SQL)
	}
}

func TestLoadDir_NonexistentDirectory(t *testing.T) {
	_, err := LoadDir("../../testdata/nonexistent")
	assertErr(t, err)
}

func TestLoadFile_NonexistentFile(t *testing.T) {
	_, err := LoadFile("../../testdata/dashboards/nonexistent.yml")
	assertErr(t, err)
}

// ---------------------------------------------------------------------------
// Auto-set tests for declarative widgets
// ---------------------------------------------------------------------------

func TestLoadFile_AutoSetsXYForDimensionalChart(t *testing.T) {
	d, err := LoadFile("../../testdata/dashboards/google-analytics.yml")
	assertNoErr(t, err)

	// Row 1 has "Daily Traffic" (dimension: daily -> event_date, metrics: [page_views, users]).
	w := d.Rows[1].Widgets[0]
	if w.Name != "Daily Traffic" {
		t.Fatalf("expected 'Daily Traffic', got %q", w.Name)
	}
	if w.X != "event_date" {
		t.Errorf("expected X = 'event_date', got %q", w.X)
	}
	if len(w.Y) != 2 || w.Y[0] != "page_views" || w.Y[1] != "users" {
		t.Errorf("expected Y = [page_views, users], got %v", w.Y)
	}
}

func TestLoadFile_AutoSetsXYForNonDateDimension(t *testing.T) {
	d, err := LoadFile("../../testdata/dashboards/google-analytics.yml")
	assertNoErr(t, err)

	// Row 2, widget 1: "Top Countries" (dimension: country -> geo.country).
	w := d.Rows[2].Widgets[1]
	if w.Name != "Top Countries" {
		t.Fatalf("expected 'Top Countries', got %q", w.Name)
	}
	// DimensionAlias("geo.country") = "country"
	if w.X != "country" {
		t.Errorf("expected X = 'country', got %q", w.X)
	}
	if len(w.Y) != 1 || w.Y[0] != "users" {
		t.Errorf("expected Y = [users], got %v", w.Y)
	}
}

func TestLoadFile_DoesNotOverrideExplicitXY(t *testing.T) {
	d, err := LoadFile("../../testdata/dashboards/google-analytics.yml")
	assertNoErr(t, err)

	// Row 2, widget 0: "Traffic Sources" — has explicit x/y, no dimension.
	w := d.Rows[2].Widgets[0]
	if w.Name != "Traffic Sources" {
		t.Fatalf("expected 'Traffic Sources', got %q", w.Name)
	}
	if w.X != "source" {
		t.Errorf("expected explicit X = 'source', got %q", w.X)
	}
}

func TestLoadFile_MetricRefColumnNotForced(t *testing.T) {
	d, err := LoadFile("../../testdata/dashboards/google-analytics.yml")
	assertNoErr(t, err)

	// Row 0, widget 0: "Page Views" — metric ref, column should be empty
	// (MetricWidget falls back to first column).
	w := d.Rows[0].Widgets[0]
	if w.Name != "Page Views" {
		t.Fatalf("expected 'Page Views', got %q", w.Name)
	}
	if w.Column != "" {
		t.Errorf("expected empty column for metric-ref widget, got %q", w.Column)
	}
}
