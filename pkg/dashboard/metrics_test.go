package dashboard

import (
	"math"
	"strings"
	"testing"
)

// --- aggregateExpr tests (all aggregate types, with and without filters) ---

func TestAggregateExpr_Count(t *testing.T) {
	t.Run("no filter", func(t *testing.T) {
		m := &Metric{Aggregate: "count"}
		got, err := aggregateExpr(m, "total")
		assertNoErr(t, err)
		assertEqual(t, got, "COUNT(*) AS total")
	})

	t.Run("with filter", func(t *testing.T) {
		m := &Metric{Aggregate: "count", Filter: map[string]string{"event_name": "page_view"}}
		got, err := aggregateExpr(m, "pv")
		assertNoErr(t, err)
		assertEqual(t, got, "COUNT(CASE WHEN event_name = 'page_view' THEN 1 END) AS pv")
	})

	t.Run("with multi-key filter", func(t *testing.T) {
		m := &Metric{Aggregate: "count", Filter: map[string]string{
			"event_name": "click",
			"platform":   "WEB",
		}}
		got, err := aggregateExpr(m, "web_clicks")
		assertNoErr(t, err)
		// Keys are sorted, so event_name comes first.
		assertContains(t, got, "event_name = 'click' AND platform = 'WEB'")
		assertContains(t, got, "COUNT(CASE WHEN")
	})
}

func TestAggregateExpr_CountDistinct(t *testing.T) {
	t.Run("no filter", func(t *testing.T) {
		m := &Metric{Aggregate: "count_distinct", Column: "user_id"}
		got, err := aggregateExpr(m, "users")
		assertNoErr(t, err)
		assertEqual(t, got, "COUNT(DISTINCT user_id) AS users")
	})

	t.Run("with filter", func(t *testing.T) {
		m := &Metric{Aggregate: "count_distinct", Column: "user_id", Filter: map[string]string{"status": "active"}}
		got, err := aggregateExpr(m, "active_users")
		assertNoErr(t, err)
		assertEqual(t, got, "COUNT(DISTINCT CASE WHEN status = 'active' THEN user_id END) AS active_users")
	})

	t.Run("missing column", func(t *testing.T) {
		m := &Metric{Aggregate: "count_distinct"}
		_, err := aggregateExpr(m, "bad")
		assertErr(t, err)
	})
}

func TestAggregateExpr_Sum(t *testing.T) {
	t.Run("no filter", func(t *testing.T) {
		m := &Metric{Aggregate: "sum", Column: "amount"}
		got, err := aggregateExpr(m, "revenue")
		assertNoErr(t, err)
		assertEqual(t, got, "SUM(amount) AS revenue")
	})

	t.Run("with filter", func(t *testing.T) {
		m := &Metric{Aggregate: "sum", Column: "amount", Filter: map[string]string{"region": "EU"}}
		got, err := aggregateExpr(m, "eu_revenue")
		assertNoErr(t, err)
		assertEqual(t, got, "SUM(CASE WHEN region = 'EU' THEN amount ELSE 0 END) AS eu_revenue")
	})

	t.Run("missing column", func(t *testing.T) {
		m := &Metric{Aggregate: "sum"}
		_, err := aggregateExpr(m, "bad")
		assertErr(t, err)
	})
}

func TestAggregateExpr_Avg(t *testing.T) {
	t.Run("no filter", func(t *testing.T) {
		m := &Metric{Aggregate: "avg", Column: "latency"}
		got, err := aggregateExpr(m, "avg_latency")
		assertNoErr(t, err)
		assertEqual(t, got, "AVG(latency) AS avg_latency")
	})

	t.Run("with filter", func(t *testing.T) {
		m := &Metric{Aggregate: "avg", Column: "latency", Filter: map[string]string{"region": "US"}}
		got, err := aggregateExpr(m, "us_avg")
		assertNoErr(t, err)
		assertEqual(t, got, "AVG(CASE WHEN region = 'US' THEN latency END) AS us_avg")
	})

	t.Run("missing column", func(t *testing.T) {
		m := &Metric{Aggregate: "avg"}
		_, err := aggregateExpr(m, "bad")
		assertErr(t, err)
	})
}

func TestAggregateExpr_MinMax(t *testing.T) {
	for _, agg := range []string{"min", "max"} {
		t.Run(agg+" no filter", func(t *testing.T) {
			m := &Metric{Aggregate: agg, Column: "price"}
			got, err := aggregateExpr(m, agg+"_price")
			assertNoErr(t, err)
			assertEqual(t, got, strings.ToUpper(agg)+"(price) AS "+agg+"_price")
		})

		t.Run(agg+" with filter", func(t *testing.T) {
			m := &Metric{Aggregate: agg, Column: "price", Filter: map[string]string{"category": "books"}}
			got, err := aggregateExpr(m, "result")
			assertNoErr(t, err)
			assertContains(t, got, strings.ToUpper(agg)+"(CASE WHEN category = 'books' THEN price END)")
		})

		t.Run(agg+" missing column", func(t *testing.T) {
			m := &Metric{Aggregate: agg}
			_, err := aggregateExpr(m, "bad")
			assertErr(t, err)
		})
	}
}

func TestAggregateExpr_UnknownAggregate(t *testing.T) {
	m := &Metric{Aggregate: "median", Column: "x"}
	_, err := aggregateExpr(m, "bad")
	assertErr(t, err)
}

// --- dateWhereClause tests ---

func TestDateWhereClause(t *testing.T) {
	t.Run("with format", func(t *testing.T) {
		s := &Source{DateColumn: "event_date", DateFormat: "%Y%m%d"}
		got := dateWhereClause(s, map[string]any{"start": "2025-01-01", "end": "2025-12-31"})
		assertEqual(t, got, "PARSE_DATE('%Y%m%d', event_date) >= '2025-01-01' AND PARSE_DATE('%Y%m%d', event_date) <= '2025-12-31'")
	})

	t.Run("without format", func(t *testing.T) {
		s := &Source{DateColumn: "created_at"}
		got := dateWhereClause(s, map[string]any{"start": "2025-01-01", "end": "2025-12-31"})
		assertEqual(t, got, "created_at >= '2025-01-01' AND created_at <= '2025-12-31'")
	})

	t.Run("no date column", func(t *testing.T) {
		s := &Source{}
		got := dateWhereClause(s, map[string]any{"start": "2025-01-01", "end": "2025-12-31"})
		assertEqual(t, got, "")
	})

	t.Run("nil filter", func(t *testing.T) {
		s := &Source{DateColumn: "d"}
		got := dateWhereClause(s, nil)
		assertEqual(t, got, "")
	})

	t.Run("missing start", func(t *testing.T) {
		s := &Source{DateColumn: "d"}
		got := dateWhereClause(s, map[string]any{"end": "2025-12-31"})
		assertEqual(t, got, "")
	})

	t.Run("missing end", func(t *testing.T) {
		s := &Source{DateColumn: "d"}
		got := dateWhereClause(s, map[string]any{"start": "2025-01-01"})
		assertEqual(t, got, "")
	})
}

// --- _TABLE_SUFFIX pruning tests ---

func TestDateWhereClause_WildcardTableSuffix(t *testing.T) {
	s := &Source{
		Table:      "`project.dataset.events_*`",
		DateColumn: "event_date",
		DateFormat: "%Y%m%d",
	}
	got := dateWhereClause(s, map[string]any{"start": "2025-06-01", "end": "2025-12-31"})
	assertContains(t, got, "_TABLE_SUFFIX BETWEEN '20250601' AND '20251231'")
}

func TestDateWhereClause_NonWildcardNoSuffix(t *testing.T) {
	s := &Source{
		Table:      "events",
		DateColumn: "event_date",
		DateFormat: "%Y%m%d",
	}
	got := dateWhereClause(s, map[string]any{"start": "2025-01-01", "end": "2025-12-31"})
	assertNotContains(t, got, "_TABLE_SUFFIX")
}

func TestTableSuffixRange(t *testing.T) {
	t.Run("YYYYMMDD", func(t *testing.T) {
		result, ok := tableSuffixRange("%Y%m%d", "2025-06-01", "2025-12-31")
		if !ok {
			t.Fatal("expected ok")
		}
		assertEqual(t, result[0], "20250601")
		assertEqual(t, result[1], "20251231")
	})

	t.Run("no format", func(t *testing.T) {
		_, ok := tableSuffixRange("", "2025-01-01", "2025-12-31")
		if ok {
			t.Fatal("expected not ok for empty format")
		}
	})

	t.Run("bad date", func(t *testing.T) {
		_, ok := tableSuffixRange("%Y%m%d", "not-a-date", "2025-12-31")
		if ok {
			t.Fatal("expected not ok for bad date")
		}
	})
}

func TestConvertDateFormat(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"%Y%m%d", "20060102"},
		{"%Y-%m-%d", "2006-01-02"},
		{"%Y", "2006"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		got := convertDateFormat(tt.in)
		if got != tt.want {
			t.Errorf("convertDateFormat(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// --- filterCondition tests ---

func TestFilterCondition(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		assertEqual(t, filterCondition(nil), "")
		assertEqual(t, filterCondition(map[string]string{}), "")
	})

	t.Run("single", func(t *testing.T) {
		got := filterCondition(map[string]string{"status": "active"})
		assertEqual(t, got, "status = 'active'")
	})

	t.Run("multiple sorted", func(t *testing.T) {
		got := filterCondition(map[string]string{"b": "2", "a": "1"})
		assertEqual(t, got, "a = '1' AND b = '2'")
	})
}

// --- GenerateMetricsSQL tests ---

func TestGenerateMetricsSQL(t *testing.T) {
	source := &Source{
		Table:      "`project.dataset.events_*`",
		DateColumn: "event_date",
		DateFormat: "%Y%m%d",
	}

	metrics := map[string]Metric{
		"page_views": {
			Aggregate: "count",
			Filter:    map[string]string{"event_name": "page_view"},
		},
		"users": {
			Aggregate: "count_distinct",
			Column:    "user_pseudo_id",
		},
		"sessions": {
			Aggregate: "count",
			Filter:    map[string]string{"event_name": "session_start"},
		},
		"pages_per_session": {
			Expression: "page_views / sessions",
		},
	}

	dateFilter := map[string]any{
		"start": "2025-01-01",
		"end":   "2025-12-31",
	}

	sql, err := GenerateMetricsSQL(source, metrics, dateFilter)
	assertNoErr(t, err)

	// Should include 3 aggregate metrics (not the expression).
	assertContains(t, sql, "COUNT(CASE WHEN event_name = 'page_view' THEN 1 END) AS page_views")
	assertContains(t, sql, "COUNT(DISTINCT user_pseudo_id) AS users")
	assertContains(t, sql, "COUNT(CASE WHEN event_name = 'session_start' THEN 1 END) AS sessions")

	// Should include FROM table.
	assertContains(t, sql, "FROM `project.dataset.events_*`")

	// Should include date filter with PARSE_DATE.
	assertContains(t, sql, "WHERE PARSE_DATE('%Y%m%d', event_date) >= '2025-01-01'")
	assertContains(t, sql, "PARSE_DATE('%Y%m%d', event_date) <= '2025-12-31'")

	// Should NOT include expression metric.
	assertNotContains(t, sql, "pages_per_session")

	// Should NOT have GROUP BY, ORDER BY, or LIMIT.
	assertNotContains(t, sql, "GROUP BY")
	assertNotContains(t, sql, "ORDER BY")
	assertNotContains(t, sql, "LIMIT")
}

func TestGenerateMetricsSQL_NoDateFormat(t *testing.T) {
	source := &Source{Table: "events", DateColumn: "created_at"}
	metrics := map[string]Metric{"total": {Aggregate: "count"}}
	dateFilter := map[string]any{"start": "2025-01-01", "end": "2025-12-31"}

	sql, err := GenerateMetricsSQL(source, metrics, dateFilter)
	assertNoErr(t, err)

	assertContains(t, sql, "created_at >= '2025-01-01'")
	assertNotContains(t, sql, "PARSE_DATE")
}

func TestGenerateMetricsSQL_NoDateColumn(t *testing.T) {
	source := &Source{Table: "events"}
	metrics := map[string]Metric{"total": {Aggregate: "count"}}

	sql, err := GenerateMetricsSQL(source, metrics, nil)
	assertNoErr(t, err)
	assertNotContains(t, sql, "WHERE")
}

func TestGenerateMetricsSQL_OnlyExpressions(t *testing.T) {
	source := &Source{Table: "events"}
	metrics := map[string]Metric{
		"ratio": {Expression: "a / b"},
	}
	_, err := GenerateMetricsSQL(source, metrics, nil)
	assertErr(t, err) // no aggregate metrics to query
}

func TestGenerateMetricsSQL_FullQuery(t *testing.T) {
	// Test exact full output for a simple case.
	source := &Source{Table: "orders", DateColumn: "date"}
	metrics := map[string]Metric{
		"revenue": {Aggregate: "sum", Column: "amount"},
	}
	dateFilter := map[string]any{"start": "2025-01-01", "end": "2025-03-01"}

	sql, err := GenerateMetricsSQL(source, metrics, dateFilter)
	assertNoErr(t, err)
	assertEqual(t, sql, "SELECT SUM(amount) AS revenue FROM orders WHERE date >= '2025-01-01' AND date <= '2025-03-01'")
}

// --- GenerateDimensionalSQL tests ---

func TestGenerateDimensionalSQL(t *testing.T) {
	source := &Source{
		Table:      "`project.dataset.events_*`",
		DateColumn: "event_date",
		DateFormat: "%Y%m%d",
	}

	metrics := map[string]Metric{
		"page_views": {
			Aggregate: "count",
			Filter:    map[string]string{"event_name": "page_view"},
		},
		"users": {
			Aggregate: "count_distinct",
			Column:    "user_pseudo_id",
		},
	}

	dateFilter := map[string]any{"start": "2025-01-01", "end": "2025-12-31"}

	t.Run("date dimension", func(t *testing.T) {
		dim := &Dimension{Column: "event_date", Type: "date"}
		sql, err := GenerateDimensionalSQL(source, metrics, []string{"page_views", "users"}, dim, dateFilter, 0)
		assertNoErr(t, err)
		assertContains(t, sql, "PARSE_DATE('%Y%m%d', event_date) AS event_date")
		assertContains(t, sql, "GROUP BY 1 ORDER BY 1")
		assertNotContains(t, sql, "LIMIT")
	})

	t.Run("non-date dimension with limit", func(t *testing.T) {
		dim := &Dimension{Column: "geo.country"}
		sql, err := GenerateDimensionalSQL(source, metrics, []string{"users"}, dim, dateFilter, 10)
		assertNoErr(t, err)
		assertContains(t, sql, "geo.country AS country")
		assertContains(t, sql, "ORDER BY 2 DESC")
		assertContains(t, sql, "LIMIT 10")
	})

	t.Run("non-date dimension column not parsed", func(t *testing.T) {
		dim := &Dimension{Column: "geo.country"}
		sql, err := GenerateDimensionalSQL(source, metrics, []string{"users"}, dim, dateFilter, 0)
		assertNoErr(t, err)
		// geo.country should NOT be wrapped in PARSE_DATE
		assertNotContains(t, sql, "PARSE_DATE('%Y%m%d', geo.country)")
	})

	t.Run("date dimension without date_format", func(t *testing.T) {
		src := &Source{Table: "events", DateColumn: "created_at"}
		dim := &Dimension{Column: "created_at", Type: "date"}
		sql, err := GenerateDimensionalSQL(src, metrics, []string{"users"}, dim, nil, 0)
		assertNoErr(t, err)
		// No date format, so column used directly.
		assertContains(t, sql, "created_at AS created_at")
		assertNotContains(t, sql, "PARSE_DATE")
	})

	t.Run("expression metric inlined", func(t *testing.T) {
		metricsWithExpr := map[string]Metric{
			"page_views": {Aggregate: "count", Filter: map[string]string{"event_name": "page_view"}},
			"sessions":   {Aggregate: "count", Filter: map[string]string{"event_name": "session_start"}},
			"pps":        {Expression: "page_views / sessions"},
		}
		dim := &Dimension{Column: "event_date", Type: "date"}
		sql, err := GenerateDimensionalSQL(source, metricsWithExpr, []string{"pps"}, dim, nil, 0)
		assertNoErr(t, err)
		// Should inline the expression with NULLIF for divide-by-zero safety.
		assertContains(t, sql, "NULLIF(")
		assertContains(t, sql, "AS pps")
		// Should still have GROUP BY.
		assertContains(t, sql, "GROUP BY 1")
	})

	t.Run("mix of aggregate and expression metrics", func(t *testing.T) {
		metricsWithExpr := map[string]Metric{
			"page_views": {Aggregate: "count", Filter: map[string]string{"event_name": "page_view"}},
			"sessions":   {Aggregate: "count", Filter: map[string]string{"event_name": "session_start"}},
			"pps":        {Expression: "page_views / sessions"},
		}
		dim := &Dimension{Column: "event_date", Type: "date"}
		sql, err := GenerateDimensionalSQL(source, metricsWithExpr, []string{"page_views", "pps"}, dim, nil, 0)
		assertNoErr(t, err)
		// Should have both the aggregate and the expression.
		assertContains(t, sql, "AS page_views")
		assertContains(t, sql, "AS pps")
	})

	t.Run("unknown metric rejected", func(t *testing.T) {
		dim := &Dimension{Column: "event_date", Type: "date"}
		_, err := GenerateDimensionalSQL(source, metrics, []string{"nonexistent"}, dim, nil, 0)
		assertErr(t, err)
	})

	t.Run("no metrics", func(t *testing.T) {
		dim := &Dimension{Column: "event_date", Type: "date"}
		_, err := GenerateDimensionalSQL(source, metrics, nil, dim, nil, 0)
		assertErr(t, err)
	})

	t.Run("full query exact match", func(t *testing.T) {
		src := &Source{Table: "sales", DateColumn: "date"}
		m := map[string]Metric{
			"revenue": {Aggregate: "sum", Column: "amount"},
		}
		dim := &Dimension{Column: "date", Type: "date"}
		df := map[string]any{"start": "2025-01-01", "end": "2025-03-01"}
		sql, err := GenerateDimensionalSQL(src, m, []string{"revenue"}, dim, df, 0)
		assertNoErr(t, err)
		assertEqual(t, sql, "SELECT date AS date, SUM(amount) AS revenue FROM sales WHERE date >= '2025-01-01' AND date <= '2025-03-01' GROUP BY 1 ORDER BY 1")
	})

	t.Run("full query non-date with limit", func(t *testing.T) {
		src := &Source{Table: "sales"}
		m := map[string]Metric{
			"revenue": {Aggregate: "sum", Column: "amount"},
		}
		dim := &Dimension{Column: "region"}
		sql, err := GenerateDimensionalSQL(src, m, []string{"revenue"}, dim, nil, 5)
		assertNoErr(t, err)
		assertEqual(t, sql, "SELECT region AS region, SUM(amount) AS revenue FROM sales GROUP BY 1 ORDER BY 2 DESC LIMIT 5")
	})

	t.Run("multiple metrics with filters", func(t *testing.T) {
		src := &Source{Table: "events"}
		m := map[string]Metric{
			"clicks":    {Aggregate: "count", Filter: map[string]string{"event_name": "click"}},
			"impressions": {Aggregate: "count", Filter: map[string]string{"event_name": "impression"}},
		}
		dim := &Dimension{Column: "campaign"}
		sql, err := GenerateDimensionalSQL(src, m, []string{"clicks", "impressions"}, dim, nil, 0)
		assertNoErr(t, err)
		assertContains(t, sql, "COUNT(CASE WHEN event_name = 'click' THEN 1 END) AS clicks")
		assertContains(t, sql, "COUNT(CASE WHEN event_name = 'impression' THEN 1 END) AS impressions")
		assertContains(t, sql, "campaign AS campaign")
		assertContains(t, sql, "GROUP BY 1")
	})
}

// --- DimensionAlias tests ---

func TestDimensionAlias(t *testing.T) {
	tests := []struct {
		dim  string
		want string
	}{
		{"event_date", "event_date"},
		{"geo.country", "country"},
		{"device.category", "category"},
		{"a.b.c", "c"},
		{"simple", "simple"},
	}
	for _, tt := range tests {
		if got := DimensionAlias(tt.dim); got != tt.want {
			t.Errorf("DimensionAlias(%q) = %q, want %q", tt.dim, got, tt.want)
		}
	}
}

// --- expressionToSQL tests ---

func TestExpressionToSQL(t *testing.T) {
	metrics := map[string]Metric{
		"page_views": {Aggregate: "count", Filter: map[string]string{"event_name": "page_view"}},
		"sessions":   {Aggregate: "count", Filter: map[string]string{"event_name": "session_start"}},
		"users":      {Aggregate: "count_distinct", Column: "user_pseudo_id"},
		"revenue":    {Aggregate: "sum", Column: "amount"},
	}

	t.Run("simple division", func(t *testing.T) {
		sql, err := expressionToSQL("page_views / sessions", metrics)
		assertNoErr(t, err)
		// Should have NULLIF wrapping the divisor.
		assertContains(t, sql, "NULLIF(")
		assertContains(t, sql, ", 0)")
		// Should have the page_views aggregate.
		assertContains(t, sql, "COUNT(CASE WHEN event_name = 'page_view' THEN 1 END)")
	})

	t.Run("addition", func(t *testing.T) {
		sql, err := expressionToSQL("page_views + sessions", metrics)
		assertNoErr(t, err)
		assertContains(t, sql, "COUNT(CASE WHEN event_name = 'page_view' THEN 1 END)")
		assertContains(t, sql, "+")
		assertContains(t, sql, "COUNT(CASE WHEN event_name = 'session_start' THEN 1 END)")
		// No NULLIF for addition.
		assertNotContains(t, sql, "NULLIF")
	})

	t.Run("multiplication with literal", func(t *testing.T) {
		sql, err := expressionToSQL("revenue * 100", metrics)
		assertNoErr(t, err)
		assertContains(t, sql, "SUM(amount)")
		assertContains(t, sql, "* 100")
	})

	t.Run("division by literal", func(t *testing.T) {
		sql, err := expressionToSQL("page_views / 100", metrics)
		assertNoErr(t, err)
		assertContains(t, sql, "NULLIF(100, 0)")
	})

	t.Run("parenthesized divisor", func(t *testing.T) {
		sql, err := expressionToSQL("page_views / (sessions + users)", metrics)
		assertNoErr(t, err)
		assertContains(t, sql, "NULLIF(")
	})

	t.Run("unknown metric", func(t *testing.T) {
		_, err := expressionToSQL("unknown / sessions", metrics)
		assertErr(t, err)
	})
}

// --- rawAggregateExpr tests ---

func TestRawAggregateExpr(t *testing.T) {
	t.Run("count no filter", func(t *testing.T) {
		m := &Metric{Aggregate: "count"}
		got, err := rawAggregateExpr(m)
		assertNoErr(t, err)
		assertEqual(t, got, "COUNT(*)")
	})

	t.Run("count with filter", func(t *testing.T) {
		m := &Metric{Aggregate: "count", Filter: map[string]string{"event_name": "click"}}
		got, err := rawAggregateExpr(m)
		assertNoErr(t, err)
		assertEqual(t, got, "COUNT(CASE WHEN event_name = 'click' THEN 1 END)")
	})

	t.Run("sum", func(t *testing.T) {
		m := &Metric{Aggregate: "sum", Column: "amount"}
		got, err := rawAggregateExpr(m)
		assertNoErr(t, err)
		assertEqual(t, got, "SUM(amount)")
	})
}

// --- EvaluateExpression tests ---

func TestEvaluateExpression(t *testing.T) {
	values := map[string]float64{
		"page_views": 1000,
		"sessions":   200,
		"users":      150,
	}

	tests := []struct {
		expr string
		want float64
	}{
		{"page_views / sessions", 5.0},
		{"page_views + users", 1150.0},
		{"page_views - users", 850.0},
		{"page_views * 2", 2000.0},
		{"(page_views + users) / sessions", 5.75},
		{"-page_views", -1000.0},
		{"100", 100.0},
		{"3.14", 3.14},
		{"page_views / 0", 0.0},     // div by zero returns 0
		{"0 / 0", 0.0},              // 0/0 returns 0
		{"page_views * 0", 0.0},
		{"(page_views)", 1000.0},     // redundant parens
		{"((page_views))", 1000.0},
		{"page_views + users + sessions", 1350.0},              // left-to-right
		{"page_views - users - sessions", 650.0},                // left-to-right subtraction
		{"2 * 3 + 4", 10.0},                                    // precedence: * before +
		{"2 + 3 * 4", 14.0},                                    // precedence: * before +
		{"(2 + 3) * 4", 20.0},                                  // parens override
		{"page_views / sessions * users", 750.0},                // left-to-right: (1000/200)*150
		{"-page_views + users", -850.0},                         // unary minus precedence
		{"-(page_views + users)", -1150.0},                      // unary minus on group
		{"100 * page_views / (page_views + users)", 86.956521739}, // complex expression
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := EvaluateExpression(tt.expr, values)
			assertNoErr(t, err)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("got %f, want %f", got, tt.want)
			}
		})
	}
}

func TestEvaluateExpression_Errors(t *testing.T) {
	values := map[string]float64{"x": 1}

	tests := []struct {
		name string
		expr string
	}{
		{"unknown metric", "unknown_metric"},
		{"trailing operator", "x +"},
		{"unclosed paren", "(x"},
		{"empty expression", ""},
		{"dangling close paren", "x)"},
		{"invalid character", "x @ y"},
		{"double operator", "x ++ 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EvaluateExpression(tt.expr, values)
			if err == nil {
				t.Errorf("expected error for %q, got nil", tt.expr)
			}
		})
	}
}

// --- AggregateMetricNames / ExpressionMetrics tests ---

func TestAggregateMetricNames(t *testing.T) {
	metrics := map[string]Metric{
		"b_metric":     {Aggregate: "count"},
		"a_metric":     {Aggregate: "sum", Column: "amount"},
		"c_expression": {Expression: "a_metric + b_metric"},
	}

	names := AggregateMetricNames(metrics)
	if len(names) != 2 {
		t.Fatalf("expected 2 aggregate metrics, got %d", len(names))
	}
	if names[0] != "a_metric" || names[1] != "b_metric" {
		t.Errorf("expected sorted [a_metric, b_metric], got %v", names)
	}
}

func TestExpressionMetrics(t *testing.T) {
	metrics := map[string]Metric{
		"a":    {Aggregate: "count"},
		"b":    {Aggregate: "sum", Column: "x"},
		"ratio": {Expression: "a / b"},
		"delta": {Expression: "a - b"},
	}

	names := ExpressionMetrics(metrics)
	if len(names) != 2 {
		t.Fatalf("expected 2 expression metrics, got %d", len(names))
	}
	if names[0] != "delta" || names[1] != "ratio" {
		t.Errorf("expected sorted [delta, ratio], got %v", names)
	}
}

// --- helpers ---

func assertNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertErr(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func assertEqual(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("got:\n  %s\nwant:\n  %s", got, want)
	}
}

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
