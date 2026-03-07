package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Registry manages built-in and user-defined themes.
type Registry struct {
	themes map[string]Theme
}

// NewRegistry creates a registry with built-in themes.
func NewRegistry() *Registry {
	r := &Registry{
		themes: map[string]Theme{
			"bruin":      BruinLight,
			"bruin-dark": BruinDark,
		},
	}
	return r
}

// LoadUserThemes loads *.yml files from the given themes directory.
// User themes can extend a built-in theme; missing tokens are inherited.
func (r *Registry) LoadUserThemes(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading themes directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yml" && ext != ".yaml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading theme %s: %w", entry.Name(), err)
		}

		var t Theme
		if err := yaml.Unmarshal(data, &t); err != nil {
			return fmt.Errorf("parsing theme %s: %w", entry.Name(), err)
		}

		// Apply inheritance from base theme.
		if t.Extends != "" {
			base, ok := r.themes[t.Extends]
			if !ok {
				return fmt.Errorf("theme %q extends unknown theme %q", t.Name, t.Extends)
			}
			merged := make(map[string]string)
			for k, v := range base.Tokens {
				merged[k] = v
			}
			for k, v := range t.Tokens {
				merged[k] = v
			}
			t.Tokens = merged
		}

		r.themes[t.Name] = t
	}

	return nil
}

// Add registers a theme in the registry.
func (r *Registry) Add(t Theme) {
	r.themes[t.Name] = t
}

// Get returns a theme by name.
func (r *Registry) Get(name string) (Theme, bool) {
	t, ok := r.themes[name]
	return t, ok
}

// List returns all available theme names.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.themes))
	for name := range r.themes {
		names = append(names, name)
	}
	return names
}
