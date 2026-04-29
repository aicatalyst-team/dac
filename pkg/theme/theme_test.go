package theme

import (
	"os"
	"path/filepath"
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

func TestLoadFile_DefaultsThemeSchemaToV1(t *testing.T) {
	path := filepath.Join(t.TempDir(), "theme.yml")
	if err := os.WriteFile(path, []byte("name: missing-schema\ntokens:\n  background: '#fff'\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	tm, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load theme without schema: %v", err)
	}
	if tm.Schema != "https://getbruin.com/schemas/dac/theme/v1" {
		t.Fatalf("expected default schema v1, got %q", tm.Schema)
	}
}
