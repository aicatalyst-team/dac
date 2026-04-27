package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bruin-data/dac/pkg/dashboard"
)

func TestInitCommand_CreatesStarterProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "starter")

	output := captureStdout(t, func() {
		if err := runInit(dir, "starter", false); err != nil {
			t.Fatalf("init failed: %v", err)
		}
	})

	for _, path := range []string{
		".bruin.yml",
		"README.md",
		"data/.gitkeep",
		"dashboards/sales.yml",
		"dashboards/semantic-sales.yml",
		"semantic/sales.yml",
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}

	dashboards, err := dashboard.LoadDir(dir)
	if err != nil {
		t.Fatalf("loading generated dashboards: %v", err)
	}
	if len(dashboards) != 2 {
		t.Fatalf("expected 2 dashboards, got %d", len(dashboards))
	}
	if err := dashboard.ValidateAll(dashboards); err != nil {
		t.Fatalf("generated dashboards should validate: %v", err)
	}
	if !strings.Contains(output, "dac serve --dir . --open") {
		t.Fatalf("expected next steps in output, got %q", output)
	}
}

func TestInitCommand_SQLTemplateSkipsSemanticFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sql")

	if err := runInit(dir, "sql", false); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "dashboards/sales.yml")); err != nil {
		t.Fatalf("expected SQL dashboard: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "semantic")); !os.IsNotExist(err) {
		t.Fatalf("expected semantic directory to be absent, got err=%v", err)
	}
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	if strings.Contains(string(readme), "Semantic Sales") {
		t.Fatalf("sql template README should not reference semantic dashboard: %s", readme)
	}
}

func TestInitCommand_TemplatesAliases(t *testing.T) {
	cases := map[string]string{
		"basic-yaml":    "sql",
		"semantic-yml":  "semantic",
		"semantic-yaml": "semantic",
		"semantic-tsx":  "tsx",
		"yaml":          "sql",
	}

	for input, want := range cases {
		if got := normalizeInitTemplate(input); got != want {
			t.Fatalf("normalizeInitTemplate(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestInitCommand_RefusesOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".bruin.yml"), []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	err := runInit(dir, "starter", false)
	if err == nil {
		t.Fatal("expected overwrite conflict")
	}
	if !strings.Contains(err.Error(), "would be overwritten") {
		t.Fatalf("expected overwrite error, got %v", err)
	}
}

func TestInitCommand_ForceOverwritesScaffoldFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".bruin.yml")
	if err := os.WriteFile(path, []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	if err := runInit(dir, "starter", true); err != nil {
		t.Fatalf("init --force failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(data), "existing") {
		t.Fatalf("expected config to be overwritten, got %q", data)
	}
}

func TestInitCommand_RegisteredWithApp(t *testing.T) {
	app := NewApp(BuildInfo{Version: "test"})
	var found bool
	for _, command := range app.Commands {
		if command.Name == "init" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected init command to be registered")
	}
}

func TestInitCommand_RunsThroughCLI(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cli")
	app := NewApp(BuildInfo{Version: "test"})

	if err := app.Run(context.Background(), []string{"dac", "init", "--template", "semantic", dir}); err != nil {
		t.Fatalf("cli init failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "dashboards/semantic-sales.yml")); err != nil {
		t.Fatalf("expected semantic dashboard: %v", err)
	}
}
