package semantic

import (
	"strings"
	"testing"
)

func TestLoadDir_FixtureModels(t *testing.T) {
	models, err := LoadDir("../../testdata/project/semantic")
	if err != nil {
		t.Fatal(err)
	}

	model, ok := models["sales"]
	if !ok {
		t.Fatalf("expected sales model, got %v", Names(models))
	}
	if model.Source.Table != "analytics.orders" {
		t.Fatalf("unexpected source table: %s", model.Source.Table)
	}
}

func TestGenerateSQL_FixtureModel(t *testing.T) {
	model, err := LoadFile("../../testdata/project/semantic/sales.yml")
	if err != nil {
		t.Fatal(err)
	}

	engine, err := NewEngine(model)
	if err != nil {
		t.Fatal(err)
	}

	sql, err := engine.GenerateSQL(&Query{
		Dimensions: []DimensionRef{{Name: "order_date", Granularity: "month"}},
		Metrics:    []string{"avg_order_value"},
		Filters: []Filter{{
			Dimension: "country",
			Operator:  "equals",
			Value:     "US",
		}},
		Segments: []string{"completed"},
		Sort: []SortSpec{{
			Name:      "order_date",
			Direction: "asc",
		}},
		Limit: 12,
	})
	if err != nil {
		t.Fatal(err)
	}

	expectContains(t, sql, "date_trunc('month', order_date) AS order_date")
	expectContains(t, sql, "sum(amount) / NULLIF(count(distinct order_id), 0) AS avg_order_value")
	expectContains(t, sql, "FROM analytics.orders")
	expectContains(t, sql, "WHERE country = 'US' AND status = 'completed'")
	expectContains(t, sql, "GROUP BY 1")
	expectContains(t, sql, "ORDER BY order_date ASC")
	expectContains(t, sql, "LIMIT 12")
}

func expectContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected SQL to contain %q\nSQL: %s", want, got)
	}
}
