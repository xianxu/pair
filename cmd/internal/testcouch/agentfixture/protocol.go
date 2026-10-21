package agentfixture

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const MaxCommandBytes = 16 << 10
const MaxOutputBytes = 32 << 10
const MaxRequests = 128
const MaxConcurrentCLI = 4
const DefaultCLITimeout = 15 * time.Second
const MaxCLITimeout = 24 * time.Hour
const OwnershipFile = ".testcouch-owner"

// ErrMailboxFull is terminal for this bounded fixture incarnation.
var ErrMailboxFull = errors.New("fixture mailbox limit reached")

// A reserved single-slot shutdown channel cannot be exhausted by ordinary work.
const shutdownFile = "shutdown.json"
const shutdownResultFile = "shutdown-result.json"

type CommandKind string

const (
	Exec     CommandKind = "exec"
	SetState CommandKind = "state"
	Stop     CommandKind = "stop"
)

type Command struct {
	Timeout time.Duration `json:"timeout_ns,omitempty"`
	PID     int           `json:"pid,omitempty"`
	RunID   string        `json:"run_id,omitempty"`
	ID      string        `json:"id"`
	Kind    CommandKind   `json:"kind"`
	Args    []string      `json:"args,omitempty"`
	State   Phase         `json:"state,omitempty"`
}
type Result struct {
	ID        string `json:"id"`
	Stdout    string `json:"stdout,omitempty"`
	Stderr    string `json:"stderr,omitempty"`
	ExitCode  int    `json:"exit_code"`
	Error     string `json:"error,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}

var requestID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,79}$`)

func ValidateCommand(c Command) error {
	if c.Timeout < 0 || c.Timeout > MaxCLITimeout || (c.Timeout != 0 && c.Kind != Exec) {
		return errors.New("invalid fixture CLI timeout")
	}
	if c.PID < 0 || len(c.RunID) > 128 {
		return errors.New("invalid fixture actor generation")
	}
	if !requestID.MatchString(c.ID) {
		return errors.New("invalid fixture request ID")
	}
	switch c.Kind {
	case Exec:
		if c.State != "" || len(c.Args) > 64 {
			return errors.New("invalid exec request")
		}
		n := 0
		for _, a := range c.Args {
			n += len(a)
			if strings.ContainsRune(a, 0) {
				return errors.New("NUL in command argument")
			}
		}
		if n > 8<<10 {
			return errors.New("command arguments exceed limit")
		}
	case SetState:
		if len(c.Args) != 0 || (c.State != Idle && c.State != Working && c.State != Picker) {
			return errors.New("invalid fixture state request")
		}
	case Stop:
		if len(c.Args) != 0 || c.State != "" {
			return errors.New("invalid stop request")
		}
	default:
		return errors.New("unknown fixture command")
	}
	return nil
}
func DecodeCommand(raw []byte) (Command, error) {
	var c Command
	if len(raw) > MaxCommandBytes {
		return c, errors.New("fixture command exceeds limit")
	}
	if err := strictjson.RejectDuplicateKeys(raw); err != nil {
		return c, err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return c, err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return c, errors.New("trailing fixture command data")
	}
	return c, ValidateCommand(c)
}

// ActorDirectory selects a mailbox from the canonical actual worktree. It is
// not a Couch identity, and cannot grant authority to issue a public command.
func ActorDirectory(base, worktree string) string {
	digest := sha256.Sum256([]byte(worktree))
	return filepath.Join(base, hex.EncodeToString(digest[:16]))
}

func WriteCommand(control string, c Command) error {
	if err := ValidateCommand(c); err != nil {
		return err
	}
	for _, sub := range []string{"commands", "results"} {
		if err := privateDirectory(filepath.Join(control, sub)); err != nil {
			return err
		}
	}
	for _, sub := range []string{"commands", "results"} {
		if c.Kind == Stop {
			break
		}
		entries, err := os.ReadDir(filepath.Join(control, sub))
		if err != nil {
			return err
		}
		if len(entries) >= MaxRequests {
			return ErrMailboxFull
		}
	}
	if _, err := os.Lstat(filepath.Join(control, "results", c.ID+".json")); err == nil {
		return errors.New("fixture request ID already completed")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	readyRaw, err := readBounded(filepath.Join(control, "ready.json"), 4096)
	if err != nil {
		return err
	}
	var ready Ready
	if err = json.Unmarshal(readyRaw, &ready); err != nil {
		return err
	}
	if ready.PID <= 0 || ready.RunID == "" {
		return errors.New("fixture actor is not ready")
	}
	if c.PID == 0 {
		c.PID = ready.PID
	}
	if c.RunID == "" {
		c.RunID = ready.RunID
	}
	if c.Kind == Stop {
		return atomicJSON(filepath.Join(control, shutdownFile), c, true)
	}
	return atomicJSON(filepath.Join(control, "commands", c.ID+".json"), c, true)
}
func ReadResult(control, id string) (Result, error) {
	var result Result
	if !requestID.MatchString(id) {
		return result, errors.New("invalid fixture request ID")
	}
	raw, err := readBounded(filepath.Join(control, "results", id+".json"), 12*MaxOutputBytes+4096)
	reserved := errors.Is(err, os.ErrNotExist)
	if reserved {
		raw, err = readBounded(filepath.Join(control, shutdownResultFile), 4096)
	}
	if err != nil {
		return result, err
	}
	if err = json.Unmarshal(raw, &result); err != nil {
		return result, err
	}
	if result.ID != id {
		if reserved {
			return Result{}, os.ErrNotExist
		}
		return Result{}, errors.New("fixture result correlation mismatch")
	}
	return result, nil
}
func atomicJSON(path string, v any, exclusive bool) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".publish-")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(append(raw, '\n')); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if exclusive {
		return os.Link(name, path)
	}
	return os.Rename(name, path)
}
func readBounded(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.New("fixture file is not bounded regular data")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err == nil && int64(len(raw)) > limit {
		err = errors.New("fixture file exceeds limit")
	}
	return raw, err
}
func privateDirectory(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("fixture directory is not a physical directory")
	}
	return nil
}
