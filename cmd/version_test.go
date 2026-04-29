package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

func TestVersionCommand_PrintsVersionAndCommit(t *testing.T) {
	cmd := versionCmd(BuildInfo{
		Version: "v1.2.3",
		Commit:  "abc123",
	})

	output := captureStdout(t, func() {
		if err := cmd.Action(context.Background(), cmd); err != nil {
			t.Fatalf("version action returned error: %v", err)
		}
	})

	if strings.TrimSpace(output) != "v1.2.3 (abc123)" {
		t.Fatalf("unexpected output %q", output)
	}
}

func TestNewApp_SetsVersionMetadata(t *testing.T) {
	app := NewApp(BuildInfo{
		Version: "v9.9.9",
		Commit:  "deadbeef",
	})

	if app.Version != "v9.9.9" {
		t.Fatalf("expected app version to be set, got %q", app.Version)
	}

	var found bool
	for _, command := range app.Commands {
		if command.Name == "version" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected version command to be registered")
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	defer r.Close()

	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read stdout: %v", err)
	}

	return buf.String()
}
