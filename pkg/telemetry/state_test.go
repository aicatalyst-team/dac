package telemetry

import (
	"testing"
	"time"

	"github.com/spf13/afero"
)

func TestLoadOrCreateInstallStateWithFS_CreatesAndPersists(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	homeDir := "/home/test/.dac"
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	state, isNew, err := loadOrCreateInstallStateWithFS(fs, homeDir, "1.2.3", func() time.Time {
		return now
	})
	if err != nil {
		t.Fatalf("first load returned error: %v", err)
	}
	if !isNew {
		t.Fatal("expected isNew=true on first load")
	}
	if state.InstallID == "" {
		t.Fatal("expected non-empty install id")
	}
	if state.InstallVersion != "1.2.3" {
		t.Fatalf("install version: got %q want %q", state.InstallVersion, "1.2.3")
	}
	if got, want := state.InstallAt, now.Format(time.RFC3339); got != want {
		t.Fatalf("install at: got %q want %q", got, want)
	}

	state2, isNew2, err := loadOrCreateInstallStateWithFS(fs, homeDir, "9.9.9", time.Now)
	if err != nil {
		t.Fatalf("second load returned error: %v", err)
	}
	if isNew2 {
		t.Fatal("expected isNew=false on second load")
	}
	if state2.InstallID != state.InstallID {
		t.Fatalf("install id changed across loads: got %q want %q", state2.InstallID, state.InstallID)
	}
	if state2.InstallAt != state.InstallAt {
		t.Fatalf("install at changed across loads: got %q want %q", state2.InstallAt, state.InstallAt)
	}
	if state2.InstallVersion != state.InstallVersion {
		t.Fatalf("install version changed across loads: got %q want %q", state2.InstallVersion, state.InstallVersion)
	}
}
