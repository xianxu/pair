package diagnosticlog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

// ErrInspectionUnavailable distinguishes incomplete native evidence from a
// complete observation that blocks one log. Scheduled sweeps must propagate
// acquisition failures instead of skipping every candidate as path-local.
var ErrInspectionUnavailable = errors.New("diagnostic inspection unavailable or incomplete")

func inspectionFailure(err error) error {
	return fmt.Errorf("%w: %w", ErrInspectionUnavailable, err)
}

type Runtime struct {
	Process  Registration
	Protocol bool
	Owner    *artifactpath.StorageOwner
	Label    string
}

// Inspection is a stateful OS seam: both exact open holders and transient
// legacy runtime writers must be known before a rename or unlink.
type Inspection interface {
	OpenFiles(ctx context.Context, path string) ([]Registration, error)
	Runtimes(ctx context.Context) ([]Runtime, error)
}

func VerifyWriters(ctx context.Context, path string, registered []Registration, probe Inspection) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if probe == nil {
		return inspectionFailure(ErrUnknownWriters)
	}
	known := func(p Registration) bool {
		for _, r := range registered {
			if r.PID == p.PID && r.Birth == p.Birth && p.Birth != "" {
				return true
			}
		}
		return false
	}
	holders, e := probe.OpenFiles(ctx, path)
	if e != nil {
		return fmt.Errorf("diagnostic open-file inspection: %w", inspectionFailure(e))
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(holders) != 0 {
		return fmt.Errorf("%w: diagnostic payload has live open holders", ErrUnknownWriters)
	}
	runtimes, e := probe.Runtimes(ctx)
	if e != nil {
		return fmt.Errorf("diagnostic runtime inspection: %w", inspectionFailure(e))
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, runtime := range runtimes {
		if err := ctx.Err(); err != nil {
			return err
		}
		if known(runtime.Process) || runtime.Protocol {
			continue
		}
		if runtime.Owner != nil {
			p, err := artifactpath.ResolveScoped(runtime.Owner.Directory(), runtime.Owner.Tag)
			if err != nil {
				return ErrUnknownWriters
			}
			if path != p.WrapEvents() && path != p.AdaptLog() && path != p.FullscreenDiagnostics() {
				continue
			}
			label := runtime.Label
			if label == "" {
				label = "Pair runtime"
			}
			return fmt.Errorf("%w: %s tag %s (pid %d): relaunch to register", ErrUnknownWriters, label, runtime.Owner.Tag, runtime.Process.PID)
		}
		sameRoot := len(registered) == 0 || runtime.Process.Root == ""
		for _, r := range registered {
			if r.Root == "" || r.Root == runtime.Process.Root {
				sameRoot = true
			}
		}
		if sameRoot {
			return fmt.Errorf("%w: legacy runtime pid %d has no matching protocol registration", ErrUnknownWriters, runtime.Process.PID)
		}
	}
	return nil
}

// OSInspection deliberately retains data when lsof/native process inspection is unavailable or denies
// inspection. It reads process metadata only; no trace contents are inspected.
type OSInspection struct {
	processes       processInspection
	openFileCommand func(context.Context, ...string) ([]byte, []byte, error)
}

type processInspection interface {
	Snapshot(context.Context) ([]procutil.Observation, error)
	Identity(string) string
	StrictIdentity(string) string
	Environment(int) (map[string]string, error)
}
type nativeProcessInspection struct{}

func (nativeProcessInspection) Snapshot(ctx context.Context) ([]procutil.Observation, error) {
	return procutil.Snapshot(ctx)
}
func (nativeProcessInspection) Identity(pid string) string       { return procutil.Identity(pid) }
func (nativeProcessInspection) StrictIdentity(pid string) string { return procutil.StrictIdentity(pid) }
func (nativeProcessInspection) Environment(pid int) (map[string]string, error) {
	return processEnvironment(pid)
}

var runtimeExecutableNames = []string{"pair", "pair-go", "pair-wrap", "pair-slug", "pair-session-watch", "pair-title-poller", "couch", "nvim", "zellij"}

func isRuntimeExecutable(name string) bool {
	for _, candidate := range runtimeExecutableNames {
		if name == candidate {
			return true
		}
	}
	return false
}

func possibleRuntimeName(short string) bool {
	// Kernel comm can be truncated. Unknown names also cannot exclude a writer.
	for _, candidate := range runtimeExecutableNames {
		if strings.HasPrefix(candidate, short) {
			return true
		}
	}
	return false
}

func command(parent context.Context, name string, args ...string) ([]byte, []byte, error) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, e := cmd.Output()
	if ctx.Err() != nil {
		return nil, stderr.Bytes(), ctx.Err()
	}
	return b, stderr.Bytes(), e
}
func (OSInspection) OpenFiles(ctx context.Context, path string) ([]Registration, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, e := os.Lstat(path); os.IsNotExist(e) {
		return nil, nil
	} else if e != nil {
		return nil, e
	}
	out, stderr, e := command(ctx, "lsof", "-nP", "-F", "p", "--", path)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if e != nil {
		var exit *exec.ExitError
		if !(errors.As(e, &exit) && exit.ExitCode() == 1 && len(out) == 0 && len(stderr) == 0) {
			return nil, errors.New("lsof unavailable or incomplete")
		}
	}
	if len(stderr) != 0 {
		return nil, errors.New("lsof reported incomplete inspection")
	}
	var result []Registration
	for _, line := range strings.Split(string(out), "\n") {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if line == "" {
			continue
		}
		if line[0] == 'f' && len(result) > 0 {
			continue
		}
		if line[0] != 'p' {
			return nil, errors.New("unexpected lsof output")
		}
		pid, e := strconv.Atoi(line[1:])
		if e != nil || pid <= 0 {
			return nil, ErrUnknownWriters
		}
		birth := procutil.StrictIdentity(strconv.Itoa(pid))
		if birth == "" {
			return nil, ErrUnknownWriters
		}
		result = append(result, Registration{PID: pid, Birth: birth})
	}
	return result, nil
}
func (o OSInspection) Runtimes(ctx context.Context) ([]Runtime, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	reader := o.processes
	if reader == nil {
		reader = nativeProcessInspection{}
	}
	rows, err := reader.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	var result []Runtime
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if row.Zombie || procutil.IsProcessGone(row.Err) {
			continue
		}
		if row.Err != nil {
			if possibleRuntimeName(row.Name) {
				return nil, fmt.Errorf("%w: runtime pid %d process metadata unavailable", ErrUnknownWriters, row.PID)
			}
			continue
		}
		// Linux appends this marker when an executable is replaced/unlinked;
		// that still-running generation remains a possible writer.
		base := strings.TrimSuffix(filepath.Base(row.Executable), " (deleted)")
		if !isRuntimeExecutable(base) {
			continue
		}
		pid := row.PID
		birth := reader.StrictIdentity(strconv.Itoa(pid))
		if birth == "" || row.Identity == "" || reader.Identity(strconv.Itoa(pid)) != row.Identity {
			return nil, fmt.Errorf("%w: runtime pid %d generation changed during inventory", ErrUnknownWriters, pid)
		}

		if err := ctx.Err(); err != nil {
			return nil, err
		}
		env, e := reader.Environment(pid)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if procutil.IsProcessGone(e) {
			continue
		}
		if e != nil || reader.StrictIdentity(strconv.Itoa(pid)) != birth {
			return nil, fmt.Errorf("%w: runtime pid %d environment unavailable or generation changed", ErrUnknownWriters, pid)
		}
		root := env["PAIR_DATA_DIR"]
		protocol := pid == os.Getpid() || env["PAIR_RETENTION_PROTOCOL"] == "1"
		// Protocol-aware writers publish their registrations under the mutation
		// lock. VerifyWriters exempts these transient runtimes regardless of root;
		// resolving an obsolete root here would block retention everywhere.
		if root != "" && !protocol {
			if filepath.Base(filepath.Dir(root)) == "repos" {
				root = filepath.Dir(filepath.Dir(root))
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			canonicalRoot, e := filepath.EvalSymlinks(root)
			if e != nil {
				return nil, fmt.Errorf("%w: runtime pid %d storage root unavailable", ErrUnknownWriters, pid)
			}
			root = canonicalRoot
		}

		// An unrelated editor or Zellij instance without Pair context cannot write
		// Pair diagnostic files through these emitters.
		if root == "" && (base == "nvim" || base == "zellij") {
			continue
		}
		owner, label, ownerErr := runtimeDiagnosticOwner(env)
		if !protocol && ownerErr != nil {
			return nil, fmt.Errorf("%w: runtime pid %d has conflicting or incomplete ownership", ErrUnknownWriters, pid)
		}
		result = append(result, Runtime{Process: Registration{PID: pid, Birth: birth, Root: root}, Protocol: protocol, Owner: owner, Label: label})
	}
	return result, nil
}
func DefaultProof(ctx context.Context, path string, writers []Registration) error {
	return VerifyWriters(ctx, path, writers, OSInspection{})
}

// Couch publishes the selected root and composite thread address before the
// outer Pair client has a PAIR_DATA_DIR. Missing protocol is not missing owner.
func runtimeDiagnosticOwner(env map[string]string) (*artifactpath.StorageOwner, string, error) {
	tag, scope := env["PAIR_TAG"], env["PAIR_SCOPE_KEY"]
	hostedTag, hostedScope := env["COUCH_THREAD_TAG"], env["COUCH_THREAD_SCOPE"]
	if (hostedTag == "") != (hostedScope == "") || tag != "" && hostedTag != "" && tag != hostedTag || scope != "" && hostedScope != "" && scope != hostedScope {
		return nil, "", ErrUnknownWriters
	}
	if tag == "" {
		tag = hostedTag
	}
	if scope == "" {
		scope = hostedScope
	}
	if tag == "" {
		return nil, "", nil
	}
	root := env["PAIR_DATA_DIR"]
	if root != "" {
		if filepath.Base(filepath.Dir(root)) == "repos" {
			actualScope := filepath.Base(root)
			if scope != "" && scope != actualScope {
				return nil, "", ErrUnknownWriters
			}
			scope = actualScope
			root = filepath.Dir(filepath.Dir(root))
		} else if scope != "" {
			return nil, "", ErrUnknownWriters
		}
	} else {
		root = env["COUCH_PAIR_DATA_DIR"]
	}
	if root == "" {
		return nil, "", ErrUnknownWriters
	}
	physical, err := filepath.EvalSymlinks(root)
	if err != nil || physical != root {
		return nil, "", ErrUnknownWriters
	}
	owner, err := artifactpath.NewStorageOwner(root, scope, tag)
	if err != nil {
		return nil, "", ErrUnknownWriters
	}
	if physical, err := filepath.EvalSymlinks(owner.Directory()); err == nil && physical != owner.Directory() {
		return nil, "", ErrUnknownWriters
	} else if err != nil && !os.IsNotExist(err) {
		return nil, "", ErrUnknownWriters
	}
	label := "Pair runtime"
	if hostedTag != "" {
		repo := filepath.Base(env["COUCH_TREE"])
		if _, err := artifactpath.NewStorageOwner(root, "", repo); err == nil {
			label = repo + " slot"
		} else {
			label = "Couch slot"
		}
	}
	return &owner, label, nil
}
