package theme

import (
	"fmt"
	"os"

	"github.com/bruin-data/dac/schemas"
	"gopkg.in/yaml.v3"
)

// Theme represents a dashboard theme with design tokens.
type Theme struct {
	Schema  string            `yaml:"schema,omitempty" json:"schema,omitempty"`
	Name    string            `yaml:"name" json:"name"`
	Extends string            `yaml:"extends,omitempty" json:"extends,omitempty"`
	Tokens  map[string]string `yaml:"tokens" json:"tokens"`
}

// LoadFile reads a single theme YAML file.
func LoadFile(path string) (Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Theme{}, fmt.Errorf("reading theme file %s: %w", path, err)
	}
	if err := schemas.ValidateYAML(schemas.ThemeV1ID, data); err != nil {
		return Theme{}, fmt.Errorf("validating theme file %s: %w", path, err)
	}
	var t Theme
	if err := yaml.Unmarshal(data, &t); err != nil {
		return Theme{}, fmt.Errorf("parsing theme file %s: %w", path, err)
	}
	if t.Schema == "" {
		t.Schema = schemas.ThemeV1ID
	}
	if t.Name == "" {
		return Theme{}, fmt.Errorf("theme file %s: missing 'name' field", path)
	}
	return t, nil
}

// BruinLight is the default "bruin" theme.
var BruinLight = Theme{
	Schema: schemas.ThemeV1ID,
	Name:   "bruin",
	Tokens: map[string]string{
		"background":     "#FFFFFF",
		"surface":        "#F6F7F9",
		"surface-hover":  "#EDEEF1",
		"border":         "#E2E4E9",
		"text-primary":   "#0A0D14",
		"text-secondary": "#525866",
		"text-muted":     "#868C98",
		"accent":         "#4338CA",
		"accent-hover":   "#3730A3",
		"accent-subtle":  "rgba(67, 56, 202, 0.06)",
		"success":        "#059669",
		"warning":        "#D97706",
		"error":          "#DC2626",
		"chart-1":        "#4338CA",
		"chart-2":        "#0891B2",
		"chart-3":        "#7C3AED",
		"chart-4":        "#DB2777",
		"chart-5":        "#CA8A04",
		"chart-6":        "#059669",
		"chart-7":        "#DC2626",
		"chart-8":        "#2563EB",
	},
}

// BruinDark is the dark variant of the "bruin" theme.
var BruinDark = Theme{
	Schema: schemas.ThemeV1ID,
	Name:   "bruin-dark",
	Tokens: map[string]string{
		"background":     "#0B0E14",
		"surface":        "#141720",
		"surface-hover":  "#1C202B",
		"border":         "#272D3B",
		"text-primary":   "#E8ECF2",
		"text-secondary": "#8892A4",
		"text-muted":     "#545E72",
		"accent":         "#7C6EF6",
		"accent-hover":   "#6D5DE6",
		"accent-subtle":  "rgba(124, 110, 246, 0.10)",
		"success":        "#34D399",
		"warning":        "#FBBF24",
		"error":          "#F87171",
		"chart-1":        "#7C6EF6",
		"chart-2":        "#22D3EE",
		"chart-3":        "#A78BFA",
		"chart-4":        "#F472B6",
		"chart-5":        "#FBBF24",
		"chart-6":        "#34D399",
		"chart-7":        "#FB7185",
		"chart-8":        "#60A5FA",
	},
}
