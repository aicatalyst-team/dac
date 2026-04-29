package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFile_ValidatesThemeSchema(t *testing.T) {
	tm, err := LoadFile("../../testdata/themes/corporate.yml")
	if err != nil {
		t.Fatalf("load theme: %v", err)
	}
	if tm.Schema == "" {
		t.Fatal("expected theme schema to be populated")
	}
	if tm.Name != "corporate" {
		t.Fatalf("expected corporate theme, got %q", tm.Name)
	}
}

func TestLoadFile_RejectsThemeWithoutSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "theme.yml")
	if err := os.WriteFile(path, []byte("name: missing-schema\ntokens:\n  background: '#fff'\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected schema validation error")
	}
	if !strings.Contains(err.Error(), "schema validation failed") {
		t.Fatalf("expected schema validation error, got %v", err)
	}
}
