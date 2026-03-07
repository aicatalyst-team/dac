package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadDir discovers and loads all *.yml dashboard files in the given directory.
func LoadDir(dir string) ([]*Dashboard, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading dashboard directory: %w", err)
	}

	var dashboards []*Dashboard
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !isYAMLFile(entry.Name()) {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		d, err := LoadFile(path)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", entry.Name(), err)
		}
		dashboards = append(dashboards, d)
	}

	return dashboards, nil
}

// LoadFile loads a single dashboard YAML file.
func LoadFile(path string) (*Dashboard, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var d Dashboard
	if err := yaml.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	d.FilePath = path

	// Resolve file-based queries (both named and inline on widgets).
	dir := filepath.Dir(path)
	if err := resolveQueryFiles(&d, dir); err != nil {
		return nil, err
	}

	// Auto-set fields for declarative widgets so the frontend works unchanged.
	for i, row := range d.Rows {
		for j, w := range row.Widgets {
			// Metric-ref: auto-set column.
			if w.MetricRef != "" && w.Column == "" {
				d.Rows[i].Widgets[j].Column = "value"
			}
			// Dimensional chart: auto-set x and y from dimension/metrics.
			if w.Dimension != "" && len(w.MetricRefs) > 0 {
				if dim, ok := d.Dimensions[w.Dimension]; ok {
					if w.X == "" {
						d.Rows[i].Widgets[j].X = DimensionAlias(dim.Column)
					}
				}
				if len(w.Y) == 0 {
					d.Rows[i].Widgets[j].Y = w.MetricRefs
				}
			}
		}
	}

	return &d, nil
}

// resolveQueryFiles reads external .sql files referenced by queries and widgets.
func resolveQueryFiles(d *Dashboard, baseDir string) error {
	// Resolve named queries with file references.
	for name, q := range d.Queries {
		if q.File != "" {
			sql, err := readQueryFile(baseDir, q.File)
			if err != nil {
				return fmt.Errorf("query %q: %w", name, err)
			}
			q.SQL = sql
			d.Queries[name] = q
		}
	}

	// Resolve inline file references on widgets.
	for i, row := range d.Rows {
		for j, w := range row.Widgets {
			if w.File != "" && w.QueryRef == "" {
				sql, err := readQueryFile(baseDir, w.File)
				if err != nil {
					return fmt.Errorf("widget %q: %w", w.Name, err)
				}
				d.Rows[i].Widgets[j].SQL = sql
			}
		}
	}

	return nil
}

func readQueryFile(baseDir, relPath string) (string, error) {
	path := filepath.Join(baseDir, relPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading query file %s: %w", relPath, err)
	}
	return string(data), nil
}

func isYAMLFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yml" || ext == ".yaml"
}
