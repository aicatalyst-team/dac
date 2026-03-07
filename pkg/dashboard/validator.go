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
			case "metric":
				errs = append(errs, validateMetricWidget(prefix, &w)...)
			case "chart":
				errs = append(errs, validateChartWidget(prefix, &w)...)
			case "table":
				// Table widgets just need a query source.
				errs = append(errs, validateQuerySource(prefix, &w, d)...)
			case "text":
				if w.Content == "" {
					errs = append(errs, fmt.Sprintf("%s: content is required for text widgets", prefix))
				}
			case "":
				// Already reported above.
			default:
				errs = append(errs, fmt.Sprintf("%s: unknown widget type %q (expected metric, chart, table, or text)", prefix, w.Type))
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

func validateMetricWidget(prefix string, w *Widget) []string {
	var errs []string
	if w.Column == "" {
		errs = append(errs, fmt.Sprintf("%s: column is required for metric widgets", prefix))
	}
	return errs
}

func validateChartWidget(prefix string, w *Widget) []string {
	var errs []string
	validCharts := map[string]bool{"line": true, "bar": true, "area": true, "pie": true}
	if w.Chart == "" {
		errs = append(errs, fmt.Sprintf("%s: chart type is required (line, bar, area, pie)", prefix))
	} else if !validCharts[w.Chart] {
		errs = append(errs, fmt.Sprintf("%s: unknown chart type %q", prefix, w.Chart))
	}

	if w.Chart == "pie" {
		if w.Label == "" {
			errs = append(errs, fmt.Sprintf("%s: label is required for pie charts", prefix))
		}
		if w.Value == "" {
			errs = append(errs, fmt.Sprintf("%s: value is required for pie charts", prefix))
		}
	} else if w.Chart != "" {
		if w.X == "" {
			errs = append(errs, fmt.Sprintf("%s: x is required for %s charts", prefix, w.Chart))
		}
		if len(w.Y) == 0 {
			errs = append(errs, fmt.Sprintf("%s: y is required for %s charts", prefix, w.Chart))
		}
	}
	return errs
}
