package dashboard

import (
	"fmt"
	"strings"
)

// ValidationError holds all validation issues for a dashboard.
type ValidationError struct {
	Dashboard string
	Errors    []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("dashboard %q has %d validation error(s):\n  - %s",
		e.Dashboard, len(e.Errors), strings.Join(e.Errors, "\n  - "))
}

// Validate checks a dashboard definition for correctness.
func Validate(d *Dashboard) error {
	var errs []string

	if d.Name == "" {
		errs = append(errs, "name is required")
	}

	if len(d.Rows) == 0 {
		errs = append(errs, "at least one row is required")
	}

	for i, row := range d.Rows {
		if len(row.Widgets) == 0 {
			errs = append(errs, fmt.Sprintf("row %d: at least one widget is required", i+1))
			continue
		}

		totalCols := 0
		for j, w := range row.Widgets {
			prefix := fmt.Sprintf("row %d, widget %d (%q)", i+1, j+1, w.Name)

			if w.Name == "" {
				errs = append(errs, fmt.Sprintf("row %d, widget %d: name is required", i+1, j+1))
			}

			if w.Type == "" {
				errs = append(errs, fmt.Sprintf("%s: type is required", prefix))
			}

			// Validate widget type.
			switch w.Type {
			case WidgetTypeMetric:
				errs = append(errs, validateMetricWidget(prefix, &w, d)...)
			case WidgetTypeChart:
				errs = append(errs, validateChartWidget(prefix, &w, d)...)
			case WidgetTypeTable:
				// Table widgets just need a query source.
				errs = append(errs, validateQuerySource(prefix, &w, d)...)
			case WidgetTypeText:
				if w.Content == "" {
					errs = append(errs, fmt.Sprintf("%s: content is required for text widgets", prefix))
				}
			case WidgetTypeDivider:
				// No required fields.
			case WidgetTypeImage:
				if w.Src == "" {
					errs = append(errs, fmt.Sprintf("%s: src is required for image widgets", prefix))
				}
			case "":
				// Already reported above.
			default:
				errs = append(errs, fmt.Sprintf("%s: unknown widget type %q (expected metric, chart, table, text, divider, or image)", prefix, w.Type))
			}

			if w.Col < 0 || w.Col > 12 {
				errs = append(errs, fmt.Sprintf("%s: col must be between 1 and 12, got %d", prefix, w.Col))
			}
			if w.Col > 0 {
				totalCols += w.Col
			}
		}

		if totalCols > 12 {
			errs = append(errs, fmt.Sprintf("row %d: total column span is %d, exceeds 12", i+1, totalCols))
		}
	}

	// Validate source and metrics.
	if len(d.Metrics) > 0 && d.Source == nil {
		errs = append(errs, "source is required when metrics are defined")
	}
	if d.Source != nil && d.Source.Table == "" {
		errs = append(errs, "source: table is required")
	}
	for name, m := range d.Metrics {
		prefix := fmt.Sprintf("metric %q", name)
		if m.Expression != "" {
			if m.Aggregate != "" {
				errs = append(errs, fmt.Sprintf("%s: cannot specify both aggregate and expression", prefix))
			}
		} else {
			if m.Aggregate == "" {
				errs = append(errs, fmt.Sprintf("%s: one of aggregate or expression is required", prefix))
			} else if !ValidAggregates[m.Aggregate] {
				errs = append(errs, fmt.Sprintf("%s: unknown aggregate %q", prefix, m.Aggregate))
			}
			if m.Aggregate != "count" && m.Column == "" {
				errs = append(errs, fmt.Sprintf("%s: column is required for %s", prefix, m.Aggregate))
			}
		}
	}
	// Validate expression references.
	for name, m := range d.Metrics {
		if m.Expression == "" {
			continue
		}
		if err := validateExpressionRefs(m.Expression, d.Metrics); err != nil {
			errs = append(errs, fmt.Sprintf("metric %q: %s", name, err.Error()))
		}
	}

	// Validate dimensions.
	if len(d.Dimensions) > 0 && d.Source == nil {
		errs = append(errs, "source is required when dimensions are defined")
	}
	for name, dim := range d.Dimensions {
		prefix := fmt.Sprintf("dimension %q", name)
		if dim.Column == "" {
			errs = append(errs, fmt.Sprintf("%s: column is required", prefix))
		}
		if dim.Type != "" && dim.Type != "date" {
			errs = append(errs, fmt.Sprintf("%s: unknown type %q (expected \"date\" or empty)", prefix, dim.Type))
		}
	}

	// Validate filters.
	for i, f := range d.Filters {
		prefix := fmt.Sprintf("filter %d (%q)", i+1, f.Name)
		if f.Name == "" {
			errs = append(errs, fmt.Sprintf("filter %d: name is required", i+1))
		}
		if f.Type == "" {
			errs = append(errs, fmt.Sprintf("%s: type is required", prefix))
		}
		validTypes := map[string]bool{"date-range": true, "select": true, "text": true}
		if f.Type != "" && !validTypes[f.Type] {
			errs = append(errs, fmt.Sprintf("%s: unknown filter type %q", prefix, f.Type))
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Dashboard: d.Name, Errors: errs}
	}
	return nil
}

func validateQuerySource(prefix string, w *Widget, d *Dashboard) []string {
	var errs []string
	if w.MetricRef != "" {
		// Metric-ref widgets get their query from the metrics system.
		return errs
	}
	if w.QueryRef == "" && w.SQL == "" && w.File == "" {
		errs = append(errs, fmt.Sprintf("%s: one of query, sql, or file is required", prefix))
	}
	if w.QueryRef != "" {
		if _, ok := d.Queries[w.QueryRef]; !ok {
			errs = append(errs, fmt.Sprintf("%s: query %q not found in queries map", prefix, w.QueryRef))
		}
	}
	return errs
}

func validateExpressionRefs(expr string, metrics map[string]Metric) error {
	// Extract identifiers from the expression and check they exist in metrics.
	pos := 0
	for pos < len(expr) {
		ch := rune(expr[pos])
		if ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			start := pos
			for pos < len(expr) {
				c := rune(expr[pos])
				if c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
					pos++
				} else {
					break
				}
			}
			name := expr[start:pos]
			if _, ok := metrics[name]; !ok {
				return fmt.Errorf("references unknown metric %q", name)
			}
		} else {
			pos++
		}
	}
	return nil
}

func validateMetricWidget(prefix string, w *Widget, d *Dashboard) []string {
	var errs []string
	if w.MetricRef != "" {
		if _, ok := d.Metrics[w.MetricRef]; !ok {
			errs = append(errs, fmt.Sprintf("%s: metric %q not found in metrics map", prefix, w.MetricRef))
		}
		return errs
	}
	if w.Column == "" {
		errs = append(errs, fmt.Sprintf("%s: column is required for metric widgets", prefix))
	}
	return errs
}

var validChartTypes = map[string]bool{
	"line": true, "bar": true, "area": true, "pie": true,
	"scatter": true, "bubble": true, "combo": true, "histogram": true,
	"boxplot": true, "funnel": true, "sankey": true, "heatmap": true,
	"calendar": true, "sparkline": true, "waterfall": true, "xmr": true,
	"dumbbell": true,
}

func validateChartWidget(prefix string, w *Widget, d *Dashboard) []string {
	var errs []string
	if w.Chart == "" {
		errs = append(errs, fmt.Sprintf("%s: chart type is required", prefix))
		return errs
	}
	if !validChartTypes[w.Chart] {
		errs = append(errs, fmt.Sprintf("%s: unknown chart type %q", prefix, w.Chart))
		return errs
	}

	// Dimensional chart: uses dimension + metrics instead of x/y/sql.
	if w.Dimension != "" || len(w.MetricRefs) > 0 {
		if w.Dimension == "" {
			errs = append(errs, fmt.Sprintf("%s: dimension is required when metrics are specified", prefix))
		} else if _, ok := d.Dimensions[w.Dimension]; !ok {
			errs = append(errs, fmt.Sprintf("%s: dimension %q not found in dimensions map", prefix, w.Dimension))
		}
		if len(w.MetricRefs) == 0 {
			errs = append(errs, fmt.Sprintf("%s: metrics are required when dimension is specified", prefix))
		}
		for _, ref := range w.MetricRefs {
			if _, ok := d.Metrics[ref]; !ok {
				errs = append(errs, fmt.Sprintf("%s: metric %q not found in metrics map", prefix, ref))
			}
		}
		return errs
	}

	switch w.Chart {
	case "pie", "funnel":
		// label + value columns
		if w.Label == "" {
			errs = append(errs, fmt.Sprintf("%s: label is required for %s charts", prefix, w.Chart))
		}
		if w.Value == "" {
			errs = append(errs, fmt.Sprintf("%s: value is required for %s charts", prefix, w.Chart))
		}

	case "sankey":
		// source + target + value columns
		if w.Source == "" {
			errs = append(errs, fmt.Sprintf("%s: source is required for sankey charts", prefix))
		}
		if w.Target == "" {
			errs = append(errs, fmt.Sprintf("%s: target is required for sankey charts", prefix))
		}
		if w.Value == "" {
			errs = append(errs, fmt.Sprintf("%s: value is required for sankey charts", prefix))
		}

	case "heatmap":
		// x + y + value
		if w.X == "" {
			errs = append(errs, fmt.Sprintf("%s: x is required for heatmap charts", prefix))
		}
		if len(w.Y) == 0 {
			errs = append(errs, fmt.Sprintf("%s: y is required for heatmap charts", prefix))
		}
		if w.Value == "" {
			errs = append(errs, fmt.Sprintf("%s: value is required for heatmap charts", prefix))
		}

	case "calendar":
		// x (date) + value
		if w.X == "" {
			errs = append(errs, fmt.Sprintf("%s: x (date column) is required for calendar charts", prefix))
		}
		if w.Value == "" {
			errs = append(errs, fmt.Sprintf("%s: value is required for calendar charts", prefix))
		}

	case "histogram":
		// x (column to bin)
		if w.X == "" {
			errs = append(errs, fmt.Sprintf("%s: x is required for histogram charts", prefix))
		}

	case "bubble":
		// x + y + size
		if w.X == "" {
			errs = append(errs, fmt.Sprintf("%s: x is required for bubble charts", prefix))
		}
		if len(w.Y) == 0 {
			errs = append(errs, fmt.Sprintf("%s: y is required for bubble charts", prefix))
		}
		if w.Size == "" {
			errs = append(errs, fmt.Sprintf("%s: size is required for bubble charts", prefix))
		}

	default:
		// line, bar, area, scatter, combo, sparkline, waterfall, xmr, dumbbell, boxplot
		// all need x + y
		if w.X == "" {
			errs = append(errs, fmt.Sprintf("%s: x is required for %s charts", prefix, w.Chart))
		}
		if len(w.Y) == 0 {
			errs = append(errs, fmt.Sprintf("%s: y is required for %s charts", prefix, w.Chart))
		}
	}

	return errs
}
