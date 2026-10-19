package historyarchive

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strings"
	"time"
	"unicode"
)

const maxFlatInventoryBytes = 64 << 20
const maxFlatInventoryEntries = 100000

type RcloneRunner interface {
	Run(context.Context, []string, int) ([]byte, error)
}

// pair:m5-concept integration
type ExecRcloneRunner struct {
	Binary string
	Env    []string
}

type boundedRcloneOutput struct {
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

func (b *boundedRcloneOutput) Write(p []byte) (int, error) {
	n := len(p)
	left := b.limit - b.buffer.Len()
	if left < len(p) {
		b.exceeded = true
		p = p[:left]
	}
	_, _ = b.buffer.Write(p)
	return n, nil
}
func (r ExecRcloneRunner) Run(ctx context.Context, args []string, limit int) ([]byte, error) {
	if limit < 1 || limit > maxFlatInventoryBytes {
		return nil, errors.New("invalid rclone output bound")
	}
	binary := r.Binary
	if binary == "" {
		binary = "rclone"
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	if r.Env != nil {
		cmd.Env = r.Env
	}
	cmd.WaitDelay = time.Second
	out := &boundedRcloneOutput{limit: limit}
	stderr := &boundedRcloneOutput{limit: 16 << 10}
	cmd.Stdout = out
	cmd.Stderr = stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if out.exceeded {
		return nil, errors.New("rclone output exceeds bound")
	}
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return nil, fmt.Errorf("rclone exited with status %d", exit.ExitCode())
		}
		return nil, errors.New("rclone unavailable")
	}
	return out.buffer.Bytes(), nil
}

// RemoteEntry is one observed object; duplicate names remain separate entries.
// pair:m5-concept pure
type RemoteEntry struct {
	Name   string
	ID     string
	Size   int64
	IsDir  bool
	Hashes map[string]string
}

func flatName(name string) bool {
	if name == "" || len(name) > 1024 || name == "." || name == ".." || strings.ContainsAny(name, "/\\") {
		return false
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
func DecodeFlatInventory(raw []byte) (map[string][]RemoteEntry, error) {
	if len(raw) > maxFlatInventoryBytes {
		return nil, errors.New("remote inventory exceeds byte bound")
	}
	var rows []json.RawMessage
	if err := decodeJSONNoDuplicateKeys(raw, &rows, false); err != nil {
		return nil, err
	}
	if rows == nil || len(rows) > maxFlatInventoryEntries {
		return nil, errors.New("invalid remote inventory count")
	}
	result := make(map[string][]RemoteEntry, len(rows))
	for _, row := range rows {
		var required map[string]json.RawMessage
		if err := json.Unmarshal(row, &required); err != nil {
			return nil, errors.New("invalid remote entry")
		}
		for key := range required {
			for _, known := range []string{"Name", "ID", "Size", "IsDir", "Hashes", "Path"} {
				if strings.EqualFold(key, known) && key != known {
					return nil, errors.New("ambiguous remote field capitalization")
				}
			}
		}
		for _, key := range []string{"Name", "Size", "IsDir"} {
			v, ok := required[key]
			if !ok || bytes.Equal(v, []byte("null")) {
				return nil, errors.New("remote entry missing required field")
			}
		}
		var entry RemoteEntry
		if err := json.Unmarshal(row, &entry); err != nil {
			return nil, errors.New("invalid remote entry fields")
		}
		if !flatName(entry.Name) || entry.Size < 0 && !entry.IsDir {
			return nil, errors.New("invalid remote entry name or size")
		}
		if p, ok := required["Path"]; ok {
			var value string
			if err := json.Unmarshal(p, &value); err != nil || value != entry.Name {
				return nil, errors.New("remote entry is not flat")
			}
		}
		result[entry.Name] = append(result[entry.Name], entry)
	}
	return result, nil
}

// Rclone runs bounded commands against one resolved destination.
// pair:m5-concept integration
type Rclone struct {
	Runner           RcloneRunner
	LocalConformance bool
}

// PinnedDrive fixes one destination folder for a pass.
// pair:m5-concept pure
type PinnedDrive struct {
	Remote   string
	FolderID string
	Root     string
}

func (r Rclone) runner() RcloneRunner {
	if r.Runner != nil {
		return r.Runner
	}
	return ExecRcloneRunner{}
}
func (r Rclone) ResolveDestination(ctx context.Context, d DriveDestination) (PinnedDrive, error) {
	parent, leaf := path.Split(d.Folder)
	parent = strings.TrimSuffix(parent, "/")
	root := d.Remote + ":"
	if parent != "" {
		root += parent
	}
	bounded, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	raw, err := r.runner().Run(bounded, []string{"lsjson", root, "--dirs-only", "--no-modtime", "--no-mimetype", "--retries", "1", "--low-level-retries", "1"}, maxFlatInventoryBytes)
	if err != nil {
		return PinnedDrive{}, err
	}
	entries, err := DecodeFlatInventory(raw)
	if err != nil {
		return PinnedDrive{}, err
	}
	matches := entries[leaf]
	if len(matches) != 1 || !matches[0].IsDir {
		return PinnedDrive{}, errors.New("Drive destination folder missing or ambiguous")
	}
	entry := matches[0]
	pin := PinnedDrive{Remote: d.Remote, FolderID: entry.ID, Root: d.Remote + ":"}
	if r.LocalConformance {
		pin.Root = d.Remote + ":" + d.Folder
	} else if !validRemoteIdentity(entry.ID) {
		return PinnedDrive{}, errors.New("Drive destination folder lacks stable ID")
	}
	return pin, nil
}
func (r Rclone) pinnedArgs(pin PinnedDrive, args ...string) []string {
	if !r.LocalConformance {
		args = append(args, "--drive-root-folder-id", pin.FolderID)
	}
	return append(args, "--retries", "1", "--low-level-retries", "1")
}
func remoteObject(pin PinnedDrive, name string) string {
	if strings.HasSuffix(pin.Root, ":") {
		return pin.Root + name
	}
	return strings.TrimSuffix(pin.Root, "/") + "/" + name
}
func (r Rclone) ObserveFlatInventory(ctx context.Context, pin PinnedDrive) (map[string][]RemoteEntry, error) {
	bounded, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	args := r.pinnedArgs(pin, "lsjson", pin.Root, "--hash", "--no-modtime", "--no-mimetype")
	raw, err := r.runner().Run(bounded, args, maxFlatInventoryBytes)
	if err != nil {
		return nil, err
	}
	return DecodeFlatInventory(raw)
}
func (r Rclone) Copy(ctx context.Context, pin PinnedDrive, localPath, name string) error {
	if !flatName(name) {
		return errors.New("invalid remote object name")
	}
	_, err := r.runner().Run(ctx, r.pinnedArgs(pin, "copyto", localPath, remoteObject(pin, name), "--immutable", "--ignore-existing", "--checksum", "--no-update-modtime"), 16<<10)
	return err
}
func (r Rclone) DownloadSHA256(ctx context.Context, pin PinnedDrive, name string) (string, error) {
	if !flatName(name) {
		return "", errors.New("invalid remote object name")
	}
	raw, err := r.runner().Run(ctx, r.pinnedArgs(pin, "hashsum", "SHA256", remoteObject(pin, name), "--download"), 4096)
	if err != nil {
		return "", err
	}
	line := strings.TrimSuffix(string(raw), "\n")
	if len(line) != 66+len(name) || line[64:66] != "  " || line[66:] != name {
		return "", errors.New("invalid downloaded checksum result")
	}
	hash := line[:64]
	if b, err := hex.DecodeString(hash); err != nil || len(b) != 32 {
		return "", errors.New("invalid downloaded checksum digest")
	}
	return strings.ToLower(hash), nil
}

var _ io.Writer = (*boundedRcloneOutput)(nil)
