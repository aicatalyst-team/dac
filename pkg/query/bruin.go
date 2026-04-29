package query

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// BruinCLIBackend implements Backend by shelling out to `bruin query`.
type BruinCLIBackend struct {
	BruinPath   string // Path to bruin binary (default: "bruin" from PATH)
	ConfigFile  string // Path to .bruin.yml
	Environment string // Target environment
}

// bruinQueryResponse mirrors the JSON output of `bruin query --output json`.
type bruinQueryResponse struct {
	Columns []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"columns"`
	Rows [][]interface{} `json:"rows"`
}

func (b *BruinCLIBackend) Execute(ctx context.Context, connection string, sql string) (*QueryResult, error) {
	bin := b.BruinPath
	if bin == "" {
		bin = "bruin"
	}

	args := []string{"query", "--output", "json"}
	if connection != "" {
		args = append(args, "-c", connection)
	}
	if b.ConfigFile != "" {
		args = append(args, "--config-file", b.ConfigFile)
	}
	if b.Environment != "" {
		args = append(args, "-e", b.Environment)
	}
	args = append(args, "-q", sql)

	cmd := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("bruin query failed: %w\nstderr: %s\nstdout: %s", err, stderr.String(), stdout.String())
	}

	var resp bruinQueryResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("parsing bruin query output: %w\nraw: %s", err, stdout.String())
	}

	result := &QueryResult{
		Rows: resp.Rows,
	}
	for _, col := range resp.Columns {
		result.Columns = append(result.Columns, ColumnInfo{
			Name: col.Name,
			Type: col.Type,
		})
	}
	return result, nil
}
