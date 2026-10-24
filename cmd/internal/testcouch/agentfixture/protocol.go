package agentfixture

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
	"golang.org/x/sys/unix"
)

const MaxCommandBytes = 16 << 10
const MaxReadyBytes = 32 << 10
const MaxOutputBytes = 32 << 10
const MaxRequests = 128
const MaxConcurrentCLI = 4
const DefaultCLITimeout = 15 * time.Second
const MaxCLITimeout = 24 * time.Hour
const OwnershipFile = ".testcouch-owner"

// ErrMailboxFull is terminal for this bounded fixture incarnation.
var ErrMailboxFull = errors.New("fixture mailbox limit reached")

// ErrCleanupUncertain requires owned fleet cleanup, not request retirement.
var ErrCleanupUncertain = errors.New("fixture CLI cleanup uncertain")

// A reserved single-slot shutdown channel cannot be exhausted by ordinary work.
const shutdownFile = "shutdown.json"
const shutdownResultFile = "shutdown-result.json"

type CommandKind string

const (
	Exec      CommandKind = "exec"
	Retain    CommandKind = "retain"
	Resources CommandKind = "resources"
	SetState  CommandKind = "state"
	Stop      CommandKind = "stop"
)

type Command struct {
	Workload         *WorkloadRequest `json:"workload,omitempty"`
	Timeout          time.Duration    `json:"timeout_ns,omitempty"`
	DeadlineUnixNano int64            `json:"deadline_unix_nano,omitempty"`
	PID              int              `json:"pid,omitempty"`
	RunID            string           `json:"run_id,omitempty"`
	Generation       string           `json:"generation,omitempty"`
	ID               string           `json:"id"`
	Kind             CommandKind      `json:"kind"`
	Args             []string         `json:"args,omitempty"`
	State            Phase            `json:"state,omitempty"`
}
type Result struct {
	Workload         *WorkloadObservation `json:"workload,omitempty"`
	ID               string               `json:"id"`
	Stdout           string               `json:"stdout,omitempty"`
	Stderr           string               `json:"stderr,omitempty"`
	ExitCode         int                  `json:"exit_code"`
	Error            string               `json:"error,omitempty"`
	Truncated        bool                 `json:"truncated,omitempty"`
	CleanupUncertain bool                 `json:"cleanup_uncertain,omitempty"`
}

func (r Result) CleanupFailure() error {
	if r.CleanupUncertain {
		return fmt.Errorf("%w: %s", ErrCleanupUncertain, r.Error)
	}
	return nil
}

var requestID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,79}$`)

func ValidateCommand(c Command) error {
	if c.Workload != nil {
		if c.Kind != SetState || c.State != Working {
			return errors.New("workload requires Working SetState")
		}
		if err := c.Workload.Validate(); err != nil {
			return err
		}
	}
	if c.DeadlineUnixNano < 0 || (c.DeadlineUnixNano != 0 && c.Kind != Exec) {
		return errors.New("invalid fixture CLI admission deadline")
	}
	if c.Timeout < 0 || c.Timeout > MaxCLITimeout || (c.Timeout != 0 && c.Kind != Exec) {
		return errors.New("invalid fixture CLI timeout")
	}
	if c.PID < 0 || len(c.RunID) > 128 || len(c.Generation) > 1024 {
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
	case Retain, Resources:
		if len(c.Args) != 0 || c.State != "" {
			return errors.New("invalid retained descendant request")
		}
	case SetState:
		if len(c.Args) != 0 || (c.State != Idle && c.State != Working && c.State != Picker && c.State != SyntheticClaudeLimit) {
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

func countMailboxEntries(entries []os.DirEntry) int {
	count := 0
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".publish-") {
			count++
		}
	}
	return count
}

// lockMailboxAdmission serializes only local validation and publication. The
// inode remains owned by the private mailbox root and is never unlinked early.
func lockMailboxAdmission(control string) (func(), error) {
	path := filepath.Join(control, ".admission.lock")
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	for {
		err = unix.Flock(fd, unix.LOCK_EX)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		break
	}
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return func() { _ = unix.Flock(fd, unix.LOCK_UN); _ = file.Close() }, nil
}

func WriteCommand(control string, c Command) error {
	if err := ValidateCommand(c); err != nil {
		return err
	}
	for _, sub := range []string{"commands", "results", "cancellations"} {
		if err := privateDirectory(filepath.Join(control, sub)); err != nil {
			return err
		}
	}
	unlock, err := lockMailboxAdmission(control)
	if err != nil {
		return err
	}
	defer unlock()
	for _, sub := range []string{"commands", "results"} {
		if c.Kind == Stop {
			break
		}
		entries, err := os.ReadDir(filepath.Join(control, sub))
		if err != nil {
			return err
		}
		if countMailboxEntries(entries) >= MaxRequests {
			return ErrMailboxFull
		}
	}
	if _, err := os.Lstat(filepath.Join(control, "results", c.ID+".json")); err == nil {
		return errors.New("fixture request ID already completed")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	readyRaw, err := readBounded(filepath.Join(control, "ready.json"), MaxReadyBytes)
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
	if c.Generation == "" {
		c.Generation = ready.Generation
	}
	if c.Kind == Exec && c.DeadlineUnixNano == 0 {
		c.DeadlineUnixNano = time.Now().Add(commandCLITimeout(c)).UnixNano()
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

// RetireResult releases a consumed immutable ordinary result. The shutdown
// result owns a separate single slot and is never removed through this API.
func RetireResult(control, id string) error {
	if !requestID.MatchString(id) {
		return errors.New("invalid fixture request ID")
	}
	unlock, err := lockMailboxAdmission(control)
	if err != nil {
		return err
	}
	defer unlock()
	path := filepath.Join(control, "results", id+".json")
	raw, err := readBounded(path, 12*MaxOutputBytes+4096)
	if err != nil {
		return err
	}
	var result Result
	if err := json.Unmarshal(raw, &result); err != nil {
		return err
	}
	if result.ID != id {
		return errors.New("fixture result correlation mismatch")
	}
	if err := result.CleanupFailure(); err != nil {
		return err
	}
	// The result fences its ID while cancellation and the original command are
	// released first. The lock prevents a new admission racing retirement.
	if err := os.Remove(filepath.Join(control, "cancellations", id+".json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// Writers can only reuse that ID once both immutable files are gone.
	if err := os.Remove(filepath.Join(control, "commands", id+".json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Remove(path)
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

// CancelCommand requests cancellation of this immutable request. Its original
// command remains the ID fence until a joined terminal result is consumed.
// Cancellation owns one marker per admitted command, outside ordinary capacity.
func CancelCommand(control, id string) error {
	if !requestID.MatchString(id) {
		return errors.New("invalid fixture request ID")
	}
	unlock, err := lockMailboxAdmission(control)
	if err != nil {
		return err
	}
	defer unlock()
	result, resultErr := ReadResult(control, id)
	if resultErr == nil && !result.CleanupUncertain {
		return nil
	}
	if resultErr != nil && !errors.Is(resultErr, os.ErrNotExist) {
		return resultErr
	}
	raw, err := readBounded(filepath.Join(control, "commands", id+".json"), MaxCommandBytes)
	if err != nil {
		return err
	}
	command, err := DecodeCommand(raw)
	if err != nil {
		return err
	}
	if command.ID != id || (command.Kind != Exec && command.Kind != SetState) {
		return errors.New("fixture cancellation request correlation/kind mismatch")
	}
	return atomicJSON(filepath.Join(control, "cancellations", id+".json"), command, false)
}

func commandCanceled(control string, command Command) (bool, error) {
	raw, err := readBounded(filepath.Join(control, "cancellations", command.ID+".json"), MaxCommandBytes)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	canceled, err := DecodeCommand(raw)
	if err != nil {
		return false, err
	}
	if canceled.ID != command.ID || canceled.Kind != command.Kind || (canceled.Kind != Exec && canceled.Kind != SetState) || canceled.PID != command.PID || canceled.RunID != command.RunID || canceled.Generation != command.Generation || canceled.DeadlineUnixNano != command.DeadlineUnixNano {
		return false, errors.New("fixture cancellation correlation mismatch")
	}
	return true, nil
}
