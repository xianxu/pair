package historyarchive

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/rowtext"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
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

// rcloneStderrTail retains final diagnostics rather than progress preceding them.
type rcloneStderrTail struct {
	data    []byte
	partial bool
}

func (b *rcloneStderrTail) Write(p []byte) (int, error) {
	const limit = 16 << 10
	n := len(p)
	if len(p) >= limit {
		offset := len(p) - limit
		if offset > 0 {
			b.partial = p[offset-1] != '\n'
		} else if len(b.data) > 0 {
			b.partial = b.data[len(b.data)-1] != '\n'
		}
		b.data = append(b.data[:0], p[offset:]...)
		return n, nil
	}
	if excess := len(b.data) + len(p) - limit; excess > 0 {
		b.partial = b.data[excess-1] != '\n'
		copy(b.data, b.data[excess:])
		b.data = b.data[:len(b.data)-excess]
	}
	b.data = append(b.data, p...)
	return n, nil
}

var rcloneCredentialValue = regexp.MustCompile(`(?i)("?(?:access[_-]?token|refresh[_-]?token|client[_-]?secret|password|passwd|api[_-]?key|token|secret)"?\s*[:=]\s*)(?:"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|[^\s,;}&]+)`)
var rcloneAuthorization = regexp.MustCompile(`(?i)(\bauthorization\s*[:=]\s*)[^\n]*`)
var rcloneBearer = regexp.MustCompile(`(?i)(\bbearer\s+)[A-Za-z0-9._~+/=-]+`)

func (b *rcloneStderrTail) diagnostic() string {
	raw := string(b.data)
	// If the retention boundary cuts through a line, discard that fragment: its
	// credential field name may have fallen outside the retained suffix.
	if b.partial {
		_, tail, ok := strings.Cut(raw, "\n")
		if !ok {
			return ""
		}
		raw = tail
	}
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		lines[i] = rowtext.Sanitize(line)
	}
	text := strings.Join(lines, "\n")
	text = rcloneCredentialValue.ReplaceAllString(text, "${1}[REDACTED]")
	text = rcloneAuthorization.ReplaceAllString(text, "${1}[REDACTED]")
	text = rcloneBearer.ReplaceAllString(text, "${1}[REDACTED]")
	if len(text) > 16<<10 {
		text = text[len(text)-(16<<10):]
		for len(text) > 0 && !utf8.RuneStart(text[0]) {
			text = text[1:]
		}
	}
	return strings.TrimSpace(text)
}
func rcloneFailure(reason string, stderr *rcloneStderrTail) error {
	if detail := stderr.diagnostic(); detail != "" {
		return fmt.Errorf("%s: %s", reason, detail)
	}
	return errors.New(reason)
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
	stderr := &rcloneStderrTail{}
	cmd.Stdout = out
	cmd.Stderr = stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if out.exceeded {
		return nil, rcloneFailure("rclone output exceeds bound", stderr)
	}
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return nil, rcloneFailure(fmt.Sprintf("rclone exited with status %d", exit.ExitCode()), stderr)
		}
		return nil, rcloneFailure("rclone unavailable", stderr)
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

// CopyBatch transfers only the coordinator's flat, privately staged objects.
func (r Rclone) CopyBatch(ctx context.Context, pin PinnedDrive, folder string) error {
	_, err := r.runner().Run(ctx, r.pinnedArgs(pin, "copy", folder, pin.Root, "--transfers", "8", "--immutable", "--ignore-existing", "--checksum", "--no-update-modtime"), 16<<10)
	return err
}

func archiveDownloadName(name string) bool {
	id := strings.TrimSuffix(name, ".tar.zst.gpg")
	if id == name {
		id = strings.TrimSuffix(name, ".json")
	}
	return id != name && validHex(id, 64) && id == strings.ToLower(id)
}

// DownloadBatch fetches only selected flat objects into caller-owned private
// staging. The exclusive filter file is removed on success and failure alike.
// Oversized objects are filtered by rclone and must be detected by the caller
// when verifying the expected staged objects.
func (r Rclone) DownloadBatch(ctx context.Context, pin PinnedDrive, names []string, destination, filterFile string, maxBytes, maxSize int64) (err error) {
	if len(names) == 0 || len(names) > maxFlatInventoryEntries || maxBytes <= 0 || maxBytes > 4<<30 || maxSize <= 0 || maxSize > 2<<30 || !filepath.IsAbs(destination) || !filepath.IsAbs(filterFile) {
		return errors.New("invalid download batch bounds or paths")
	}
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if !archiveDownloadName(name) || seen[name] {
			return errors.New("invalid or duplicate download name")
		}
		seen[name] = true
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	info, err := os.Stat(destination)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("download destination must be an existing directory")
	}
	list, err := os.OpenFile(filterFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := os.Remove(filterFile); removeErr != nil && !os.IsNotExist(removeErr) {
			err = errors.Join(err, removeErr)
		}
	}()
	for _, name := range names {
		if _, err = io.WriteString(list, "+ /"+name+"\n"); err != nil {
			return errors.Join(err, list.Close())
		}
	}
	if _, err = io.WriteString(list, "- **\n"); err != nil {
		return errors.Join(err, list.Close())
	}
	err = list.Sync()
	err = errors.Join(err, list.Close())
	if err != nil {
		return err
	}
	// rclone rejects --files-from-raw combined with --max-size. Exact rooted
	// rules retain the file-size filter and exclude every unselected object.
	args := r.pinnedArgs(pin, "copy", pin.Root, destination, "--filter-from", filterFile, "--transfers", "8", "--max-transfer", strconv.FormatInt(maxBytes, 10)+"B", "--cutoff-mode", "hard", "--max-size", strconv.FormatInt(maxSize, 10)+"B")
	_, err = r.runner().Run(ctx, args, 16<<10)
	return err
}

// DownloadSHA256Batch hashes the exact flat names in the private raw file list.
func (r Rclone) DownloadSHA256Batch(ctx context.Context, pin PinnedDrive, listPath string) (map[string]string, error) {
	raw, err := r.runner().Run(ctx, r.pinnedArgs(pin, "hashsum", "SHA256", pin.Root, "--download", "--files-from-raw", listPath, "--checkers", "8"), maxFlatInventoryBytes)
	if err != nil {
		return nil, err
	}
	if len(raw) > maxFlatInventoryBytes {
		return nil, errors.New("downloaded checksum output exceeds bound")
	}
	result := map[string]string{}
	text := strings.TrimSuffix(string(raw), "\n")
	if text == "" {
		return result, nil
	}
	for line := range strings.SplitSeq(text, "\n") {
		if len(line) < 67 || line[64:66] != "  " || !flatName(line[66:]) {
			return nil, errors.New("invalid downloaded checksum result")
		}
		hash, name := line[:64], line[66:]
		if b, err := hex.DecodeString(hash); err != nil || len(b) != 32 {
			return nil, errors.New("invalid downloaded checksum digest")
		}
		if _, exists := result[name]; exists {
			return nil, errors.New("duplicate downloaded checksum name")
		}
		if len(result) >= maxFlatInventoryEntries {
			return nil, errors.New("downloaded checksum count exceeds bound")
		}
		result[name] = strings.ToLower(hash)
	}
	return result, nil
}

var _ io.Writer = (*boundedRcloneOutput)(nil)
