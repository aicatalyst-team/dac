package query

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBruinCLIBackendDryRun(t *testing.T) {
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args.txt")
	bruinPath := filepath.Join(dir, "bruin")
	script := `#!/bin/sh
printf '%s\n' "$@" > "` + argsPath + `"
printf '{"connectionName":"warehouse","connectionType":"duckdb","query":"SELECT 1","valid":true}'
`
	if err := os.WriteFile(bruinPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bruin: %v", err)
	}

	backend := &BruinCLIBackend{
		BruinPath:   bruinPath,
		ConfigFile:  "bruin.yml",
		Environment: "dev",
	}
	result, err := backend.DryRun(context.Background(), "warehouse", "SELECT 1")
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}
	if !result.Valid || result.ConnectionName != "warehouse" || result.ConnectionType != "duckdb" {
		t.Fatalf("unexpected dry-run result: %+v", result)
	}

	data, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	got := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{
		"query",
		"--output",
		"json",
		"--dry-run",
		"-c",
		"warehouse",
		"--config-file",
		"bruin.yml",
		"-e",
		"dev",
		"-q",
		"SELECT 1",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected args:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestBruinCLIBackendDryRunReturnsBruinError(t *testing.T) {
	dir := t.TempDir()
	bruinPath := filepath.Join(dir, "bruin")
	script := `#!/bin/sh
printf '{"error":"table not found"}'
exit 1
`
	if err := os.WriteFile(bruinPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake bruin: %v", err)
	}

	backend := &BruinCLIBackend{BruinPath: bruinPath}
	_, err := backend.DryRun(context.Background(), "warehouse", "SELECT * FROM missing")
	if err == nil {
		t.Fatal("expected dry-run error")
	}
	if !strings.Contains(err.Error(), "table not found") {
		t.Fatalf("expected bruin error in message, got %v", err)
	}
}
