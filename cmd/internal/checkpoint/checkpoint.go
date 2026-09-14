// Package checkpoint owns the immutable, bounded handoff shared by the writer,
// launcher and Couch. A source path records provenance; Body is the recovery source.
package checkpoint

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

const DigestEnv = "PAIR_CONTINUATION_DIGEST"

const Version = 1
const MaxBytes = 256 * 1024

type Checkpoint struct {
	Version    int    `json:"version"`
	SourcePath string `json:"source_path"`
	Body       string `json:"body"`
	Digest     string `json:"digest"`
}

func New(sourcePath, body string) (Checkpoint, error) {
	c := Checkpoint{Version: Version, SourcePath: sourcePath, Body: body, Digest: fmt.Sprintf("%x", sha256.Sum256([]byte(body)))}
	if err := c.Validate(); err != nil {
		return Checkpoint{}, err
	}
	return c, nil
}

func (c Checkpoint) Validate() error {
	if c.Version != Version {
		return fmt.Errorf("checkpoint: unsupported version %d", c.Version)
	}
	if !filepath.IsAbs(c.SourcePath) || filepath.Clean(c.SourcePath) != c.SourcePath || len(c.SourcePath) > 4096 || !utf8.ValidString(c.SourcePath) || strings.IndexFunc(c.SourcePath, unicode.IsControl) >= 0 {
		return fmt.Errorf("checkpoint: source must be a clean absolute path without control characters")
	}
	if len(c.Body) == 0 || len(c.Body) > MaxBytes || !utf8.ValidString(c.Body) || strings.ContainsRune(c.Body, '\x00') {
		return fmt.Errorf("checkpoint: body must be nonempty UTF-8 text at most %d bytes", MaxBytes)
	}
	if FrontmatterField(c.Body, "type") != "continuation" || c.Agent() == "" {
		return fmt.Errorf("checkpoint: expected continuation frontmatter with an agent")
	}
	if !HasNextAction(c.Body) {
		return fmt.Errorf("checkpoint: missing nonempty NEXT ACTION")
	}
	if c.Digest != fmt.Sprintf("%x", sha256.Sum256([]byte(c.Body))) {
		return fmt.Errorf("checkpoint: SHA-256 digest mismatch")
	}
	return nil
}

func (c Checkpoint) Agent() string { return FrontmatterField(c.Body, "agent") }

// ReadFile bounds IO before constructing the validated snapshot. Open first and
// inspect the same handle so the regular-file check applies to the bytes read.
func ReadFile(path string) (Checkpoint, error) {
	if !filepath.IsAbs(path) {
		return Checkpoint{}, fmt.Errorf("checkpoint: source path must be absolute")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return Checkpoint{}, fmt.Errorf("read checkpoint %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return Checkpoint{}, fmt.Errorf("checkpoint: source is not a regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return Checkpoint{}, fmt.Errorf("read checkpoint %q: %w", path, err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return Checkpoint{}, err
	}
	if !st.Mode().IsRegular() {
		return Checkpoint{}, fmt.Errorf("checkpoint: source is not a regular file")
	}
	if st.Size() > MaxBytes {
		return Checkpoint{}, fmt.Errorf("checkpoint: source exceeds %d bytes", MaxBytes)
	}
	raw, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	if err != nil {
		return Checkpoint{}, fmt.Errorf("read checkpoint %q: %w", path, err)
	}
	return New(path, string(raw))
}

// FrontmatterField accepts only fields inside a closed leading YAML fence.
// Continuation fields are simple scalar lines; duplicate keys are invalid.
func FrontmatterField(body, key string) string {
	lines := strings.Split(body, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return ""
	}
	value, seen := "", false
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			return value
		}
		if strings.HasPrefix(line, key+":") {
			if seen {
				return ""
			}
			seen = true
			value = strings.TrimSpace(strings.TrimPrefix(line, key+":"))
		}
	}
	return ""
}

func firstNextActionLine(body string) (string, bool, bool) {
	lines := strings.Split(body, "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		closed := false
		for i, line := range lines[1:] {
			if strings.TrimSpace(line) == "---" {
				lines = lines[i+2:]
				closed = true
				break
			}
		}
		if !closed {
			return "", false, false
		}
	}
	for i, line := range lines {
		if strings.TrimSpace(line) != "## NEXT ACTION" {
			continue
		}
		for _, rest := range lines[i+1:] {
			t := strings.TrimSpace(rest)
			if t != "" {
				return t, IsATXHeading(t), true
			}
		}
		return "", false, false
	}
	return "", false, false
}

func HasNextAction(body string) bool {
	_, heading, found := firstNextActionLine(body)
	return found && !heading
}
func NextActionPreview(body string) string {
	line, heading, found := firstNextActionLine(body)
	if !found || heading {
		return ""
	}
	return line
}
func IsATXHeading(t string) bool {
	i := 0
	for i < len(t) && t[i] == '#' {
		i++
	}
	return i > 0 && i < len(t) && t[i] == ' '
}
