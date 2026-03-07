package dashboard

// Dashboard represents a complete dashboard definition loaded from YAML.
type Dashboard struct {
	Name        string           `yaml:"name" json:"name"`
	Description string           `yaml:"description,omitempty" json:"description,omitempty"`
	Connection  string           `yaml:"connection,omitempty" json:"connection,omitempty"`
	Theme       string           `yaml:"theme,omitempty" json:"theme,omitempty"`
	Refresh     *RefreshConfig   `yaml:"refresh,omitempty" json:"refresh,omitempty"`
	Filters     []Filter         `yaml:"filters,omitempty" json:"filters,omitempty"`
	Queries     map[string]Query `yaml:"queries,omitempty" json:"queries,omitempty"`
	Rows        []Row            `yaml:"rows" json:"rows"`

	// FilePath is the source file path, not serialized to JSON for API consumers.
	FilePath string `yaml:"-" json:"-"`
}

type RefreshConfig struct {
	Interval string `yaml:"interval" json:"interval"`
}

type Filter struct {
	Name     string         `yaml:"name" json:"name"`
	Type     string         `yaml:"type" json:"type"`
	Multiple bool           `yaml:"multiple,omitempty" json:"multiple,omitempty"`
	Default  any            `yaml:"default,omitempty" json:"default,omitempty"`
	Options  *FilterOptions `yaml:"options,omitempty" json:"options,omitempty"`
}

type FilterOptions struct {
	Values     []string `yaml:"values,omitempty" json:"values,omitempty"`
	Query      string   `yaml:"query,omitempty" json:"query,omitempty"`
	Connection string   `yaml:"connection,omitempty" json:"connection,omitempty"`
}

// Query represents a named query definition.
type Query struct {
	SQL        string `yaml:"sql,omitempty" json:"sql,omitempty"`
	File       string `yaml:"file,omitempty" json:"file,omitempty"`
	Connection string `yaml:"connection,omitempty" json:"connection,omitempty"`
}

type Row struct {
	Height  any      `yaml:"height,omitempty" json:"height,omitempty"`
	Widgets []Widget `yaml:"widgets" json:"widgets"`
}

// Widget represents a single dashboard widget.
// Query resolution priority: query (named ref) > sql (inline) > file (external).
type Widget struct {
	Name string `yaml:"name" json:"name"`
	Type string `yaml:"type" json:"type"` // metric, chart, table, text
	Col  int    `yaml:"col,omitempty" json:"col,omitempty"`

	// Query source (pick one)
	QueryRef string `yaml:"query,omitempty" json:"query,omitempty"` // reference to queries map key
	SQL      string `yaml:"sql,omitempty" json:"sql,omitempty"`
	File     string `yaml:"file,omitempty" json:"file,omitempty"`

	// Connection override for inline queries
	Connection string `yaml:"connection,omitempty" json:"connection,omitempty"`

	// Metric fields
	Column string `yaml:"column,omitempty" json:"column,omitempty"`
	Prefix string `yaml:"prefix,omitempty" json:"prefix,omitempty"`
	Suffix string `yaml:"suffix,omitempty" json:"suffix,omitempty"`
	Format string `yaml:"format,omitempty" json:"format,omitempty"`

	// Chart fields
	Chart   string   `yaml:"chart,omitempty" json:"chart,omitempty"` // line, bar, area, pie
	X       string   `yaml:"x,omitempty" json:"x,omitempty"`
	Y       []string `yaml:"y,omitempty" json:"y,omitempty"`
	Label   string   `yaml:"label,omitempty" json:"label,omitempty"`   // for pie charts
	Value   string   `yaml:"value,omitempty" json:"value,omitempty"`   // for pie charts
	Stacked bool     `yaml:"stacked,omitempty" json:"stacked,omitempty"` // for bar/area charts

	// Table fields
	Columns []TableColumn `yaml:"columns,omitempty" json:"columns,omitempty"`

	// Text fields
	Content string `yaml:"content,omitempty" json:"content,omitempty"`
}

type TableColumn struct {
	Name   string `yaml:"name" json:"name"`
	Label  string `yaml:"label,omitempty" json:"label,omitempty"`
	Format string `yaml:"format,omitempty" json:"format,omitempty"`
}

// ResolvedQuery returns the SQL and connection for this widget, resolving named query references.
func (w *Widget) ResolvedQuery(dashboard *Dashboard) (sql, connection string, err error) {
	switch {
	case w.QueryRef != "":
		q, ok := dashboard.Queries[w.QueryRef]
		if !ok {
			return "", "", &QueryNotFoundError{Name: w.QueryRef, Widget: w.Name}
		}
		conn := q.Connection
		if conn == "" {
			conn = dashboard.Connection
		}
		if q.SQL != "" {
			return q.SQL, conn, nil
		}
		// File-based query — SQL should have been loaded by the loader.
		return q.SQL, conn, nil

	case w.SQL != "":
		conn := w.Connection
		if conn == "" {
			conn = dashboard.Connection
		}
		return w.SQL, conn, nil

	case w.File != "":
		// File-based inline query — SQL should have been loaded by the loader.
		conn := w.Connection
		if conn == "" {
			conn = dashboard.Connection
		}
		return w.SQL, conn, nil

	default:
		if w.Type == "text" {
			return "", "", nil
		}
		return "", "", &NoQueryError{Widget: w.Name}
	}
}

type QueryNotFoundError struct {
	Name   string
	Widget string
}

func (e *QueryNotFoundError) Error() string {
	return "widget \"" + e.Widget + "\": query \"" + e.Name + "\" not found in queries map"
}

type NoQueryError struct {
	Widget string
}

func (e *NoQueryError) Error() string {
	return "widget \"" + e.Widget + "\": no query, sql, or file specified"
}
