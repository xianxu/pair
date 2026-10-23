package vmcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxResultManifest = 4096
const maxGuestExport int64 = maxResultManifest + 128 + 8*65536
const maxResultRecord int64 = 65536

type resultFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}
type resultManifest struct {
	Version int          `json:"version"`
	Files   []resultFile `json:"files"`
}

// decodeResultManifest admits only the bootstrap's fixed result catalog. This
// generic runner exports status/diagnostics; command stdout/stderr are host logs.
func decodeResultManifest(raw []byte, remaining int64) (resultManifest, error) {
	var m resultManifest
	refuse := func() (resultManifest, error) {
		return resultManifest{}, fmt.Errorf("invalid or oversized guest result manifest")
	}
	if len(raw) > maxResultManifest || int64(len(raw)) > remaining {
		return refuse()
	}
	fields, _, err := requiredObject(raw, map[string]bool{"version": true, "files": true})
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(fields["version"], &m.Version); err != nil || m.Version != 1 {
		return refuse()
	}
	var files []json.RawMessage
	if err := json.Unmarshal(fields["files"], &files); err != nil || len(files) == 0 || len(files) > 9 {
		return refuse()
	}
	seen := map[string]bool{}
	remaining -= int64(len(raw))
	for _, encoded := range files {
		fields, _, err := requiredObject(encoded, map[string]bool{"path": true, "size": true})
		if err != nil {
			return m, err
		}
		var entry resultFile
		if err := json.Unmarshal(fields["path"], &entry.Path); err != nil {
			return refuse()
		}
		if bytes.Equal(bytes.TrimSpace(fields["size"]), []byte("null")) {
			return refuse()
		}
		if err := json.Unmarshal(fields["size"], &entry.Size); err != nil {
			return refuse()
		}
		cap := int64(-1)
		if entry.Path == "status" {
			cap = 128
		}
		if strings.HasPrefix(entry.Path, "diagnostics/zellij-") && len(entry.Path) == len("diagnostics/zellij-0.log") {
			index := entry.Path[len("diagnostics/zellij-")]
			if index >= '0' && index <= '7' && strings.HasSuffix(entry.Path, ".log") {
				cap = 65536
			}
		}
		if cap < 0 || seen[entry.Path] || entry.Size < 0 || entry.Size > cap || entry.Size > remaining {
			return refuse()
		}
		seen[entry.Path] = true
		remaining -= entry.Size
		m.Files = append(m.Files, entry)
	}
	if !seen["status"] {
		return refuse()
	}
	return m, nil
}

type resultFetcher func(context.Context, string, int64) ([]byte, bool, error)

// exportGuestResults bounds data before every host write, preserving raw
// additive manifest members while exporting only validated catalog paths.
func exportGuestResults(ctx context.Context, guestRoot, destination string, remaining int64, fetch resultFetcher) error {
	entries, err := os.ReadDir(destination)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("guest export destination must be empty")
	}
	raw, present, err := fetch(ctx, filepath.Join(guestRoot, "manifest.json"), maxResultManifest)
	if err != nil {
		return err
	}
	if !present {
		return fmt.Errorf("guest result manifest unavailable")
	}
	manifest, err := decodeResultManifest(raw, remaining)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(destination)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := writeExportedResult(ctx, root, "manifest.json", raw, &remaining); err != nil {
		return err
	}
	for _, entry := range manifest.Files {
		if err := ctx.Err(); err != nil {
			return err
		}
		raw, present, err := fetch(ctx, filepath.Join(guestRoot, entry.Path), entry.Size)
		if err != nil {
			return err
		}
		if !present || int64(len(raw)) != entry.Size {
			return fmt.Errorf("guest result %q changed or is incomplete", entry.Path)
		}
		if err := writeExportedResult(ctx, root, entry.Path, raw, &remaining); err != nil {
			return err
		}
	}
	return nil
}
func writeExportedResult(ctx context.Context, root *os.Root, path string, raw []byte, remaining *int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if int64(len(raw)) > *remaining {
		return fmt.Errorf("guest export byte budget exceeded")
	}
	if err := root.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := root.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	n, writeErr := file.Write(raw)
	closeErr := file.Close()
	*remaining -= int64(n)
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}
