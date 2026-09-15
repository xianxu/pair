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

	"github.com/xianxu/pair/cmd/internal/procutil"
)

type Runtime struct {
	Process  Registration
	Protocol bool
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
		return ErrUnknownWriters
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
		return fmt.Errorf("diagnostic open-file inspection: %w", e)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, p := range holders {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !known(p) {
			return ErrUnknownWriters
		}
	}
	runtimes, e := probe.Runtimes(ctx)
	if e != nil {
		return fmt.Errorf("diagnostic runtime inspection: %w", e)
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
		sameRoot := len(registered) == 0 || runtime.Process.Root == ""
		for _, r := range registered {
			if r.Root == "" || r.Root == runtime.Process.Root {
				sameRoot = true
			}
		}
		if sameRoot {
			return ErrUnknownWriters
		}
	}
	return nil
}

// OSInspection deliberately retains data when lsof/ps is unavailable or denies
// inspection. It reads process metadata only; no trace contents are inspected.
type OSInspection struct{}

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
func (OSInspection) Runtimes(ctx context.Context) ([]Runtime, error) {
	out, stderr, e := command(ctx, "ps", "-axo", "pid=,comm=")
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if e != nil || len(stderr) != 0 {
		return nil, errors.New("process inventory unavailable")
	}
	var result []Runtime
	for _, line := range strings.Split(string(out), "\n") {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		base := filepath.Base(strings.Join(fields[1:], " "))
		switch base {
		case "pair", "pair-go", "pair-wrap", "pair-slug", "pair-session-watch", "pair-title-poller", "couch", "nvim", "zellij":
		default:
			continue
		}
		pid, e := strconv.Atoi(fields[0])
		if e != nil || pid <= 0 {
			return nil, ErrUnknownWriters
		}
		birth := procutil.StrictIdentity(strconv.Itoa(pid))
		if birth == "" {
			return nil, ErrUnknownWriters
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		env, e := processEnvironment(pid)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if e != nil || procutil.StrictIdentity(strconv.Itoa(pid)) != birth {
			return nil, ErrUnknownWriters
		}
		root := env["PAIR_DATA_DIR"]
		protocol := pid == os.Getpid() || env["PAIR_RETENTION_PROTOCOL"] == "1"
		if root != "" {
			if filepath.Base(filepath.Dir(root)) == "repos" {
				root = filepath.Dir(filepath.Dir(root))
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			canonicalRoot, e := filepath.EvalSymlinks(root)
			if e != nil {
				return nil, ErrUnknownWriters
			}
			root = canonicalRoot
		}

		// An unrelated editor or Zellij instance without Pair context cannot write
		// Pair diagnostic files through these emitters.
		if root == "" && (base == "nvim" || base == "zellij") {
			continue
		}
		result = append(result, Runtime{Registration{PID: pid, Birth: birth, Root: root}, protocol})
	}
	return result, nil
}
func DefaultProof(ctx context.Context, path string, writers []Registration) error {
	return VerifyWriters(ctx, path, writers, OSInspection{})
}
