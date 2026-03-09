package dashboard

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"
)

// ValidAggregates lists the supported aggregate functions.
var ValidAggregates = map[string]bool{
	"count":          true,
	"count_distinct": true,
	"sum":            true,
	"avg":            true,
	"min":            true,
	"max":            true,
}

// GenerateMetricsSQL builds a single SELECT that computes all aggregate
// (non-expression) metrics from the source table. The returned SQL may
// contain Jinja template expressions (e.g. in the table name) and should
// be rendered through the template engine before execution.
func GenerateMetricsSQL(source *Source, metrics map[string]Metric, dateFilter map[string]any) (string, error) {
	// Collect aggregate metrics in deterministic order.
	names := AggregateMetricNames(metrics)
	if len(names) == 0 {
		return "", fmt.Errorf("no aggregate metrics to query")
	}

	var selects []string
	for _, name := range names {
		m := metrics[name]
		expr, err := aggregateExpr(&m, name)
		if err != nil {
			return "", err
		}
		selects = append(selects, expr)
	}

	var sb strings.Builder
	sb.WriteString("SELECT ")
	sb.WriteString(strings.Join(selects, ", "))
	sb.WriteString(" FROM ")
	sb.WriteString(source.Table)

	if where := dateWhereClause(source, dateFilter); where != "" {
		sb.WriteString(" WHERE ")
		sb.WriteString(where)
	}

	return sb.String(), nil
}

// AggregateMetricNames returns sorted names of all non-expression metrics.
func AggregateMetricNames(metrics map[string]Metric) []string {
	var names []string
	for name, m := range metrics {
		if !m.IsExpression() {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// ExpressionMetrics returns sorted names of all expression metrics.
func ExpressionMetrics(metrics map[string]Metric) []string {
	var names []string
	for name, m := range metrics {
		if m.IsExpression() {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// GenerateDimensionalSQL builds a SELECT that computes the given metrics
// grouped by a dimension column. Used for declarative chart widgets.
func GenerateDimensionalSQL(source *Source, allMetrics map[string]Metric, metricNames []string, dim *Dimension, dateFilter map[string]any, limit int) (string, error) {
	if len(metricNames) == 0 {
		return "", fmt.Errorf("at least one metric is required for dimensional queries")
	}

	dimAlias := DimensionAlias(dim.Column)
	dimExpr := dim.Column

	// If the dimension column is the date column and a format is set, parse it.
	if dim.Column == source.DateColumn && source.DateFormat != "" {
		dimExpr = fmt.Sprintf("PARSE_DATE('%s', %s)", source.DateFormat, dim.Column)
	}

	selects := []string{fmt.Sprintf("%s AS %s", dimExpr, dimAlias)}

	for _, name := range metricNames {
		m, ok := allMetrics[name]
		if !ok {
			return "", fmt.Errorf("metric %q not found", name)
		}
		if m.IsExpression() {
			// Inline the expression as SQL by substituting metric names
			// with their raw aggregate expressions.
			sqlExpr, err := expressionToSQL(m.Expression, allMetrics)
			if err != nil {
				return "", fmt.Errorf("metric %q: %w", name, err)
			}
			selects = append(selects, fmt.Sprintf("%s AS %s", sqlExpr, name))
		} else {
			expr, err := aggregateExpr(&m, name)
			if err != nil {
				return "", err
			}
			selects = append(selects, expr)
		}
	}

	var sb strings.Builder
	sb.WriteString("SELECT ")
	sb.WriteString(strings.Join(selects, ", "))
	sb.WriteString(" FROM ")
	sb.WriteString(source.Table)

	if where := dateWhereClause(source, dateFilter); where != "" {
		sb.WriteString(" WHERE ")
		sb.WriteString(where)
	}

	sb.WriteString(" GROUP BY 1")

	if dim.IsDate() {
		sb.WriteString(" ORDER BY 1")
	} else {
		sb.WriteString(" ORDER BY 2 DESC")
	}

	if limit > 0 {
		fmt.Fprintf(&sb, " LIMIT %d", limit)
	}

	return sb.String(), nil
}

// DimensionAlias returns a clean column alias for a dimension expression.
// e.g. "geo.country" -> "country", "event_date" -> "event_date".
func DimensionAlias(dimension string) string {
	if idx := strings.LastIndex(dimension, "."); idx >= 0 {
		return dimension[idx+1:]
	}
	return dimension
}

// expressionToSQL converts an arithmetic expression over metric names into
// a SQL expression by substituting each metric name with its raw aggregate SQL.
// For example, "page_views / sessions" becomes
// "COUNT(CASE WHEN ... END) * 1.0 / NULLIF(COUNT(CASE WHEN ... END), 0)".
func expressionToSQL(expr string, allMetrics map[string]Metric) (string, error) {
	// Build a map of metric name -> raw SQL for all aggregate metrics.
	rawSQL := make(map[string]string)
	for name, m := range allMetrics {
		if m.IsExpression() {
			continue
		}
		raw, err := rawAggregateExpr(&m)
		if err != nil {
			return "", fmt.Errorf("metric %q: %w", name, err)
		}
		rawSQL[name] = raw
	}

	// Walk the expression, replacing identifiers with their SQL.
	// Division gets NULLIF wrapping on the right operand for safety.
	var result strings.Builder
	pos := 0
	for pos < len(expr) {
		ch := rune(expr[pos])
		if unicode.IsLetter(ch) || ch == '_' {
			start := pos
			for pos < len(expr) {
				c := rune(expr[pos])
				if unicode.IsLetter(c) || c == '_' || (pos > start && unicode.IsDigit(c)) {
					pos++
				} else {
					break
				}
			}
			name := expr[start:pos]
			sql, ok := rawSQL[name]
			if !ok {
				return "", fmt.Errorf("unknown metric %q in expression", name)
			}
			result.WriteString(sql)
		} else if ch == '/' {
			// Wrap the divisor in NULLIF(..., 0) for divide-by-zero safety.
			result.WriteString(" * 1.0 / NULLIF(")
			pos++ // skip '/'

			// Parse the right operand (could be a metric name, number, or parenthesized expr).
			for pos < len(expr) && expr[pos] == ' ' {
				pos++
			}
			if pos < len(expr) && expr[pos] == '(' {
				// Parenthesized sub-expression: find the matching ')'.
				depth := 0
				subStart := pos
				for pos < len(expr) {
					if expr[pos] == '(' {
						depth++
					} else if expr[pos] == ')' {
						depth--
						if depth == 0 {
							pos++
							break
						}
					}
					pos++
				}
				subExpr := expr[subStart+1 : pos-1] // strip outer parens
				subSQL, err := expressionToSQL(subExpr, allMetrics)
				if err != nil {
					return "", err
				}
				result.WriteString("(")
				result.WriteString(subSQL)
				result.WriteString(")")
			} else if pos < len(expr) && (unicode.IsLetter(rune(expr[pos])) || expr[pos] == '_') {
				// Metric name.
				start := pos
				for pos < len(expr) {
					c := rune(expr[pos])
					if unicode.IsLetter(c) || c == '_' || (pos > start && unicode.IsDigit(c)) {
						pos++
					} else {
						break
					}
				}
				name := expr[start:pos]
				sql, ok := rawSQL[name]
				if !ok {
					return "", fmt.Errorf("unknown metric %q in expression", name)
				}
				result.WriteString(sql)
			} else if pos < len(expr) && (expr[pos] >= '0' && expr[pos] <= '9' || expr[pos] == '.') {
				// Number literal.
				for pos < len(expr) && (expr[pos] >= '0' && expr[pos] <= '9' || expr[pos] == '.') {
					result.WriteByte(expr[pos])
					pos++
				}
			} else {
				return "", fmt.Errorf("unexpected character after '/' at position %d", pos)
			}
			result.WriteString(", 0)")
		} else {
			result.WriteByte(expr[pos])
			pos++
		}
	}
	return result.String(), nil
}

// rawAggregateExpr returns the aggregate SQL expression without an alias.
func rawAggregateExpr(m *Metric) (string, error) {
	hasFilter := len(m.Filter) > 0
	cond := filterCondition(m.Filter)

	switch m.Aggregate {
	case "count":
		if hasFilter {
			return fmt.Sprintf("COUNT(CASE WHEN %s THEN 1 END)", cond), nil
		}
		return "COUNT(*)", nil
	case "count_distinct":
		if m.Column == "" {
			return "", fmt.Errorf("column is required for count_distinct")
		}
		if hasFilter {
			return fmt.Sprintf("COUNT(DISTINCT CASE WHEN %s THEN %s END)", cond, m.Column), nil
		}
		return fmt.Sprintf("COUNT(DISTINCT %s)", m.Column), nil
	case "sum":
		if m.Column == "" {
			return "", fmt.Errorf("column is required for sum")
		}
		if hasFilter {
			return fmt.Sprintf("SUM(CASE WHEN %s THEN %s ELSE 0 END)", cond, m.Column), nil
		}
		return fmt.Sprintf("SUM(%s)", m.Column), nil
	case "avg":
		if m.Column == "" {
			return "", fmt.Errorf("column is required for avg")
		}
		if hasFilter {
			return fmt.Sprintf("AVG(CASE WHEN %s THEN %s END)", cond, m.Column), nil
		}
		return fmt.Sprintf("AVG(%s)", m.Column), nil
	case "min":
		if m.Column == "" {
			return "", fmt.Errorf("column is required for min")
		}
		if hasFilter {
			return fmt.Sprintf("MIN(CASE WHEN %s THEN %s END)", cond, m.Column), nil
		}
		return fmt.Sprintf("MIN(%s)", m.Column), nil
	case "max":
		if m.Column == "" {
			return "", fmt.Errorf("column is required for max")
		}
		if hasFilter {
			return fmt.Sprintf("MAX(CASE WHEN %s THEN %s END)", cond, m.Column), nil
		}
		return fmt.Sprintf("MAX(%s)", m.Column), nil
	default:
		return "", fmt.Errorf("unknown aggregate %q", m.Aggregate)
	}
}

func aggregateExpr(m *Metric, alias string) (string, error) {
	hasFilter := len(m.Filter) > 0
	cond := filterCondition(m.Filter)

	switch m.Aggregate {
	case "count":
		if hasFilter {
			return fmt.Sprintf("COUNT(CASE WHEN %s THEN 1 END) AS %s", cond, alias), nil
		}
		return fmt.Sprintf("COUNT(*) AS %s", alias), nil

	case "count_distinct":
		if m.Column == "" {
			return "", fmt.Errorf("metric %q: column is required for count_distinct", alias)
		}
		if hasFilter {
			return fmt.Sprintf("COUNT(DISTINCT CASE WHEN %s THEN %s END) AS %s", cond, m.Column, alias), nil
		}
		return fmt.Sprintf("COUNT(DISTINCT %s) AS %s", m.Column, alias), nil

	case "sum":
		if m.Column == "" {
			return "", fmt.Errorf("metric %q: column is required for sum", alias)
		}
		if hasFilter {
			return fmt.Sprintf("SUM(CASE WHEN %s THEN %s ELSE 0 END) AS %s", cond, m.Column, alias), nil
		}
		return fmt.Sprintf("SUM(%s) AS %s", m.Column, alias), nil

	case "avg":
		if m.Column == "" {
			return "", fmt.Errorf("metric %q: column is required for avg", alias)
		}
		if hasFilter {
			return fmt.Sprintf("AVG(CASE WHEN %s THEN %s END) AS %s", cond, m.Column, alias), nil
		}
		return fmt.Sprintf("AVG(%s) AS %s", m.Column, alias), nil

	case "min":
		if m.Column == "" {
			return "", fmt.Errorf("metric %q: column is required for min", alias)
		}
		if hasFilter {
			return fmt.Sprintf("MIN(CASE WHEN %s THEN %s END) AS %s", cond, m.Column, alias), nil
		}
		return fmt.Sprintf("MIN(%s) AS %s", m.Column, alias), nil

	case "max":
		if m.Column == "" {
			return "", fmt.Errorf("metric %q: column is required for max", alias)
		}
		if hasFilter {
			return fmt.Sprintf("MAX(CASE WHEN %s THEN %s END) AS %s", cond, m.Column, alias), nil
		}
		return fmt.Sprintf("MAX(%s) AS %s", m.Column, alias), nil

	default:
		return "", fmt.Errorf("metric %q: unknown aggregate %q", alias, m.Aggregate)
	}
}

func filterCondition(filter map[string]string) string {
	if len(filter) == 0 {
		return ""
	}
	// Sort keys for deterministic output.
	keys := make([]string, 0, len(filter))
	for k := range filter {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	conds := make([]string, 0, len(filter))
	for _, col := range keys {
		conds = append(conds, fmt.Sprintf("%s = '%s'", col, filter[col]))
	}
	return strings.Join(conds, " AND ")
}

func dateWhereClause(source *Source, dateFilter map[string]any) string {
	if source.DateColumn == "" || dateFilter == nil {
		return ""
	}
	start, _ := dateFilter["start"].(string)
	end, _ := dateFilter["end"].(string)
	if start == "" || end == "" {
		return ""
	}

	col := source.DateColumn
	if source.DateFormat != "" {
		col = fmt.Sprintf("PARSE_DATE('%s', %s)", source.DateFormat, source.DateColumn)
	}

	clause := fmt.Sprintf("%s >= '%s' AND %s <= '%s'", col, start, col, end)

	// For BigQuery wildcard tables (ending with *), add _TABLE_SUFFIX pruning
	// so BigQuery can skip entire shards at the storage layer instead of
	// scanning every table matching the wildcard.
	if strings.HasSuffix(source.Table, "*`") || strings.HasSuffix(source.Table, "*") {
		if suffix, ok := tableSuffixRange(source.DateFormat, start, end); ok {
			clause += fmt.Sprintf(" AND _TABLE_SUFFIX BETWEEN '%s' AND '%s'", suffix[0], suffix[1])
		}
	}

	return clause
}

// tableSuffixRange converts ISO date strings (2025-06-01) to BigQuery
// _TABLE_SUFFIX format based on the source's date_format. Returns the
// start/end suffix strings and true, or false if conversion isn't possible.
func tableSuffixRange(dateFormat, start, end string) ([2]string, bool) {
	if dateFormat == "" {
		return [2]string{}, false
	}

	startDate, err := time.Parse("2006-01-02", start)
	if err != nil {
		return [2]string{}, false
	}
	endDate, err := time.Parse("2006-01-02", end)
	if err != nil {
		return [2]string{}, false
	}

	goFmt := convertDateFormat(dateFormat)
	if goFmt == "" {
		return [2]string{}, false
	}

	return [2]string{startDate.Format(goFmt), endDate.Format(goFmt)}, true
}

// convertDateFormat converts a strftime-style format to Go's time format.
func convertDateFormat(fmt string) string {
	r := strings.NewReplacer(
		"%Y", "2006",
		"%m", "01",
		"%d", "02",
		"%H", "15",
		"%M", "04",
		"%S", "05",
	)
	result := r.Replace(fmt)
	// If nothing was replaced, the format is unsupported.
	if result == fmt {
		return ""
	}
	return result
}

// EvaluateExpression evaluates a simple arithmetic expression with metric
// name substitution. Supports +, -, *, /, parentheses, and numeric literals.
// Division by zero returns 0.
func EvaluateExpression(expr string, values map[string]float64) (float64, error) {
	p := &exprParser{input: expr, values: values}
	result, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	p.skipSpace()
	if p.pos < len(p.input) {
		return 0, fmt.Errorf("unexpected character %q at position %d", p.input[p.pos], p.pos)
	}
	if math.IsNaN(result) || math.IsInf(result, 0) {
		return 0, nil
	}
	return result, nil
}

type exprParser struct {
	input  string
	pos    int
	values map[string]float64
}

func (p *exprParser) skipSpace() {
	for p.pos < len(p.input) && p.input[p.pos] == ' ' {
		p.pos++
	}
}

// expr = term (('+' | '-') term)*
func (p *exprParser) parseExpr() (float64, error) {
	left, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpace()
		if p.pos >= len(p.input) {
			break
		}
		op := p.input[p.pos]
		if op != '+' && op != '-' {
			break
		}
		p.pos++
		right, err := p.parseTerm()
		if err != nil {
			return 0, err
		}
		if op == '+' {
			left += right
		} else {
			left -= right
		}
	}
	return left, nil
}

// term = factor (('*' | '/') factor)*
func (p *exprParser) parseTerm() (float64, error) {
	left, err := p.parseFactor()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpace()
		if p.pos >= len(p.input) {
			break
		}
		op := p.input[p.pos]
		if op != '*' && op != '/' {
			break
		}
		p.pos++
		right, err := p.parseFactor()
		if err != nil {
			return 0, err
		}
		if op == '*' {
			left *= right
		} else {
			if right == 0 {
				return 0, nil
			}
			left /= right
		}
	}
	return left, nil
}

// factor = '-'? (NUMBER | IDENT | '(' expr ')')
func (p *exprParser) parseFactor() (float64, error) {
	p.skipSpace()
	if p.pos >= len(p.input) {
		return 0, fmt.Errorf("unexpected end of expression")
	}

	// Unary minus.
	if p.input[p.pos] == '-' {
		p.pos++
		val, err := p.parseFactor()
		if err != nil {
			return 0, err
		}
		return -val, nil
	}

	// Parenthesized expression.
	if p.input[p.pos] == '(' {
		p.pos++
		val, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		p.skipSpace()
		if p.pos >= len(p.input) || p.input[p.pos] != ')' {
			return 0, fmt.Errorf("missing closing parenthesis")
		}
		p.pos++
		return val, nil
	}

	// Number.
	if p.input[p.pos] >= '0' && p.input[p.pos] <= '9' || p.input[p.pos] == '.' {
		return p.parseNumber()
	}

	// Identifier (metric name).
	name := p.parseIdent()
	if name == "" {
		return 0, fmt.Errorf("unexpected character %q at position %d", p.input[p.pos], p.pos)
	}
	val, ok := p.values[name]
	if !ok {
		return 0, fmt.Errorf("unknown metric %q in expression", name)
	}
	return val, nil
}

func (p *exprParser) parseNumber() (float64, error) {
	start := p.pos
	for p.pos < len(p.input) && (p.input[p.pos] >= '0' && p.input[p.pos] <= '9' || p.input[p.pos] == '.') {
		p.pos++
	}
	var val float64
	_, err := fmt.Sscanf(p.input[start:p.pos], "%f", &val)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q", p.input[start:p.pos])
	}
	return val, nil
}

func (p *exprParser) parseIdent() string {
	start := p.pos
	for p.pos < len(p.input) {
		ch := rune(p.input[p.pos])
		if unicode.IsLetter(ch) || ch == '_' || (p.pos > start && unicode.IsDigit(ch)) {
			p.pos++
		} else {
			break
		}
	}
	return p.input[start:p.pos]
}
