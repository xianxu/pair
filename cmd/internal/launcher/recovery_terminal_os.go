package launcher

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

type recoveryTerminalOS struct{ osSessionQuiescenceOps }

func newRecoveryTerminalOS() recoveryTerminalOS {
	return recoveryTerminalOS{newOSSessionQuiescenceOps()}
}

type recoveryLeaseSet []io.Closer

func (s recoveryLeaseSet) Close() error {
	var err error
	for i := len(s) - 1; i >= 0; i-- {
		err = errors.Join(err, s[i].Close())
	}
	return err
}
func (o recoveryTerminalOS) Acquire(ctx context.Context, session string, servers []SessionServerIdentity) (io.Closer, error) {
	root, err := ResolveZellijSocketRoot(os.Getenv("ZELLIJ_SOCKET_DIR"), os.TempDir(), os.Getenv("XDG_RUNTIME_DIR"), runtime.GOOS, os.Getuid())
	if err != nil {
		return nil, err
	}
	sockets := []string{filepath.Join(root, "contract_version_1", session)}
	for _, s := range servers {
		if !validQuiescenceServer(s, session) {
			return nil, errors.New("invalid recovery server snapshot")
		}
		sockets = append(sockets, s.Socket)
	}
	cache, err := artifactpath.ResolvePairCache(os.Getenv("HOME"))
	if err != nil {
		return nil, err
	}
	keys := map[string]bool{}
	var locks []string
	for _, socket := range sockets {
		canonical, err := canonicalRecoverySocket(socket)
		if err != nil {
			return nil, err
		}
		lock, err := cache.TerminalLease(canonical)
		if err != nil {
			return nil, err
		}
		if !keys[lock] {
			keys[lock] = true
			locks = append(locks, lock)
		}
	}
	sort.Strings(locks)
	var held recoveryLeaseSet
	for _, lock := range locks {
		lease, err := AcquireTerminalLease(ctx, lock)
		if err != nil {
			return nil, errors.Join(err, held.Close())
		}
		held = append(held, lease)
	}
	return held, nil
}

// Canonicalize existing ancestors without creating a missing socket namespace.
// A later cooperating launch's mkdir/EvalSymlinks resolves to the same lease key.
func canonicalRecoverySocket(socket string) (string, error) {
	if !filepath.IsAbs(socket) || filepath.Clean(socket) != socket {
		return "", errors.New("noncanonical recovery socket")
	}
	dir := filepath.Dir(socket)
	var missing []string
	for {
		physical, err := filepath.EvalSymlinks(dir)
		if err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				physical = filepath.Join(physical, missing[i])
			}
			return filepath.Join(physical, filepath.Base(socket)), nil
		}
		if !errors.Is(err, os.ErrNotExist) || dir == filepath.Dir(dir) {
			return "", err
		}
		missing = append(missing, filepath.Base(dir))
		dir = filepath.Dir(dir)
	}
}

type recoveryCommandOutput struct{ buffer bytes.Buffer }

func (b *recoveryCommandOutput) Len() int       { return b.buffer.Len() }
func (b *recoveryCommandOutput) String() string { return b.buffer.String() }

func (b *recoveryCommandOutput) Write(p []byte) (int, error) {
	if len(p) > 8<<20-b.Len() {
		return 0, errors.New("process command snapshot exceeds bound")
	}
	return b.buffer.Write(p)
}
func (o recoveryTerminalOS) Observe(ctx context.Context, session string) (recoveryTerminalSnapshot, error) {
	var result recoveryTerminalSnapshot
	if ctx == nil {
		return result, errors.New("process observation requires context")
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	rows, err := procutil.Snapshot(ctx)
	if err != nil {
		return result, err
	}
	after, err := procutil.Table()
	if err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	return projectRecoveryTerminalSnapshot(session, rows, after)
}

// The native snapshot already fences each argv read by process generation.
// The final table also fences parent relationships across the whole observation:
// unstable or unavailable descendant argv remains missing evidence, and a
// changed exact named server refuses the observation rather than moving authority.
func projectRecoveryTerminalSnapshot(session string, rows []procutil.Observation, after []procutil.Process) (recoveryTerminalSnapshot, error) {
	result := recoveryTerminalSnapshot{Processes: after, Commands: map[int]string{}, Args: map[int][]string{}}
	current := make(map[int]procutil.Process, len(after))
	for _, p := range after {
		if _, exists := current[p.PID]; exists {
			return result, errors.New("duplicate process identity")
		}
		current[p.PID] = p
	}
	seen := make(map[int]bool, len(rows))
	var output recoveryCommandOutput
	for _, row := range rows {
		if seen[row.PID] {
			return result, errors.New("duplicate process command")
		}
		seen[row.PID] = true
		if row.Err != nil || row.Zombie || len(row.Args) == 0 {
			continue
		}
		// Bound all successfully observed commands, including ones whose final
		// generation/parent check below prevents them from becoming authority.
		if _, err := output.Write([]byte(strconv.Itoa(row.PID) + " ")); err != nil {
			return result, err
		}
		for i, arg := range row.Args {
			if i > 0 {
				if _, err := output.Write([]byte(" ")); err != nil {
					return result, err
				}
			}
			if _, err := output.Write([]byte(arg)); err != nil {
				return result, err
			}
		}
		if _, err := output.Write([]byte("\n")); err != nil {
			return result, err
		}
		p, exists := current[row.PID]
		if !exists || p.Identity == "" || row.Process != p {
			continue
		}
		command := strings.TrimSpace(strings.Join(row.Args, " "))
		if command != "" {
			result.Commands[row.PID] = command
			result.Args[row.PID] = append([]string(nil), row.Args...)
		}
	}
	servers, err := serverProcesses(rows)
	if err != nil {
		return result, err
	}
	for _, server := range servers {
		if server.Session != session {
			continue
		}
		p, exists := current[server.PID]
		if !exists || p.Identity == "" {
			return result, errors.New("terminal generation moved during command observation")
		}
		// The command map contains only rows stable in identity and ancestry.
		if _, stable := result.Commands[server.PID]; !stable {
			return result, errors.New("terminal generation moved during command observation")
		}
		result.Servers = append(result.Servers, server)
	}
	return result, nil
}

// KillProcess rechecks the exact accepted generation and argv at the native
// signal boundary. The driver has just rechecked the whole immutable plan.
func (o recoveryTerminalOS) KillProcess(ctx context.Context, witness recoveryProcessWitness) error {
	if witness.PID <= 0 || witness.Identity == "" || len(witness.Args) == 0 {
		return errors.New("invalid exact owner signal authority")
	}
	rows, err := o.processes.Snapshot(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.PID != witness.PID {
			continue
		}
		if row.Err != nil {
			return row.Err
		}
		if row.Identity != witness.Identity || row.PPID != witness.PPID || !reflect.DeepEqual(row.Args, witness.Args) {
			return errors.New("accepted owner changed before signal")
		}
		if row.Zombie {
			return nil
		}
		if o.identity()(strconv.Itoa(witness.PID)) != witness.Identity {
			return errors.New("accepted owner generation changed before signal")
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		return o.killer()(witness.PID)
	}
	return nil // Positively absent from a successful native snapshot.
}
