package slides

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2/google"
	driveapi "google.golang.org/api/drive/v3"
	slidesapi "google.golang.org/api/slides/v1"
)

var oauthScopes = []string{
	slidesapi.PresentationsScope,
	driveapi.DriveFileScope,
}

// authorize returns Google credentials.
// It tries Application Default Credentials (gcloud) first,
// then falls back to a credentials.json file from ~/.dac/.
func authorize(ctx context.Context, credentialsOverride string) (*google.Credentials, error) {
	// Try ADC first (works if the user has run `gcloud auth application-default login`).
	creds, err := google.FindDefaultCredentials(ctx, oauthScopes...)
	if err == nil {
		return creds, nil
	}

	// Fall back to explicit credentials.json.
	credsFile, err := resolveCredentials(credentialsOverride)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(credsFile)
	if err != nil {
		return nil, fmt.Errorf("reading credentials: %w", err)
	}

	creds, err = google.CredentialsFromJSON(ctx, data, oauthScopes...)
	if err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}

	return creds, nil
}

// configDir returns ~/.dac, creating it if needed.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	dir := filepath.Join(home, ".dac")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("creating config directory: %w", err)
	}
	return dir, nil
}

// resolveCredentials finds the credentials.json to use:
// explicit flag > ~/.dac/credentials.json.
func resolveCredentials(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "credentials.json")
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("no Google credentials found\n\nRun:\n  gcloud auth application-default login\nor pass --credentials <path-to-credentials.json>")
	}
	return path, nil
}
