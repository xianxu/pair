package termcmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// termDiagnostics admits only bounded fault evidence. Screen observations are
// never records; the managed writer owns optional IO outside endpoint locks.
type termDiagnostics struct{ writer *diagnosticlog.BufferedWriter }

func openTermDiagnostics(root string) (*termDiagnostics, error) {
	if root == "" {
		return nil, errors.New("terminal diagnostic root is empty")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	opts := diagnosticlog.EnvironmentOptions(func(key string) string {
		if key == "PAIR_DATA_DIR" {
			return root
		}
		return ""
	})
	w, err := diagnosticlog.OpenBuffered(filepath.Join(root, "term-diagnostics.jsonl"), opts)
	if err != nil {
		return nil, err
	}
	return &termDiagnostics{writer: w}, nil
}

func (d *termDiagnostics) observe(o terminal.Observation) {
	if d == nil || o.Kind != "cell-degraded" || len(o.Data) > 4096 || len(o.EndpointID) > 256 || !json.Valid(o.Data) {
		return
	}
	record := struct {
		Version  int             `json:"version"`
		Event    string          `json:"event"`
		Time     time.Time       `json:"time"`
		Endpoint string          `json:"endpoint"`
		Cell     json.RawMessage `json:"cell"`
	}{1, "cell-degraded", time.Now().UTC(), o.EndpointID, o.Data}
	data, err := json.Marshal(record)
	if err == nil {
		_, _ = d.writer.Write(append(data, '\n'))
	}
}

func (d *termDiagnostics) Close() error {
	if d == nil {
		return nil
	}
	return d.writer.Close()
}
