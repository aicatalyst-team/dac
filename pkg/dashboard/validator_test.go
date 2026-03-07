package dashboard

import (
	"errors"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Validator tests
// ---------------------------------------------------------------------------

func TestValidate_ValidDashboard(t *testing.T) {
	d, err := LoadFile("../../testdata/dashboards/sales.yml")
	assertNoErr(t, err)

	err = Validate(d)
	assertNoErr(t, err)
}

func TestValidate_MissingName(t *testing.T) {
	d := &Dashboard{
		Rows: []Row{
			{Widgets: []Widget{{Name: "w", Type: WidgetTypeText, Content: "hi"}}},
		},
	}
	err := Validate(d)
	assertErr(t, err)
	assertValidationContains(t, err, "name is required")
}

func TestValidate_MissingRows(t *testing.T) {
	d := &Dashboard{Name: "test"}
	err := Validate(d)
	assertErr(t, err)
	assertValidationContains(t, err, "at least one row is required")
}

func TestValidate_EmptyRow(t *testing.T) {
	d := &Dashboard{
		Name: "test",
		Rows: []Row{
			{Widgets: []Widget{}},
		},
	}
	err := Validate(d)
	assertErr(t, err)
	assertValidationContains(t, err, "at least one widget is required")
}

func TestValidate_WidgetMissingName(t *testing.T) {
	d := &Dashboard{
		Name: "test",
		Rows: []Row{
			{Widgets: []Widget{{Type: WidgetTypeText, Content: "hi"}}},
		},
	}
	err := Validate(d)
	assertErr(t, err)
	assertValidationContains(t, err, "name is required")
}

func TestValidate_WidgetMissingType(t *testing.T) {
	d := &Dashboard{
		Name: "test",
		Rows: []Row{
			{Widgets: []Widget{{Name: "w"}}},
		},
	}
	err := Validate(d)
	assertErr(t, err)
	assertValidationContains(t, err, "type is required")
}

func TestValidate_ColumnSumExceeds12(t *testing.T) {
	d := &Dashboard{
		Name: "test",
		Rows: []Row{
			{Widgets: []Widget{
				{Name: "a", Type: WidgetTypeText, Content: "hi", Col: 8},
				{Name: "b", Type: WidgetTypeText, Content: "hi", Col: 6},
			}},
		},
	}
	err := Validate(d)
	assertErr(t, err)
	assertValidationContains(t, err, "exceeds 12")
}

func TestValidate_TextWidgetMissingContent(t *testing.T) {
	d := &Dashboard{
		Name: "test",
		Rows: []Row{
			{Widgets: []Widget{{Name: "w", Type: WidgetTypeText}}},
		},
	}
	err := Validate(d)
	assertErr(t, err)
	assertValidationContains(t, err, "content is required")
}

func TestValidate_ChartWidgetMissingChartType(t *testing.T) {
	d := &Dashboard{
		Name: "test",
		Rows: []Row{
			{Widgets: []Widget{{Name: "w", Type: WidgetTypeChart, SQL: "SELECT 1"}}},
		},
	}
	err := Validate(d)
	assertErr(t, err)
	assertValidationContains(t, err, "chart type is required")
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func assertValidationContains(t *testing.T, err error, substr string) {
	t.Helper()
	var ve *ValidationError
	if errors.As(err, &ve) {
		for _, e := range ve.Errors {
			if strings.Contains(e, substr) {
				return
			}
		}
		t.Errorf("expected validation error containing %q, got errors: %v", substr, ve.Errors)
		return
	}
	if strings.Contains(err.Error(), substr) {
		return
	}
	t.Errorf("expected error containing %q, got: %v", substr, err)
}
