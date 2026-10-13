package launcher

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	root := os.Getenv("ZELLIJ_SOCKET_DIR")
	if root == "" {
		root = filepath.Join(os.TempDir(), "zellij-"+strconv.Itoa(os.Getuid()))
	}
	if !filepath.IsAbs(root) {
		return nil, errors.New("terminal socket directory must be absolute")
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
	before, err := procutil.Table()
	if err != nil {
		return result, err
	}
	var output recoveryCommandOutput
	cmd := exec.CommandContext(ctx, "ps", "-axo", "pid=,command=")
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil {
		return result, err
	}
	after, err := procutil.Table()
	if err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	prior := map[int]procutil.Process{}
	for _, p := range before {
		if _, exists := prior[p.PID]; exists {
			return result, errors.New("duplicate process identity")
		}
		prior[p.PID] = p
	}
	current := map[int]procutil.Process{}
	for _, p := range after {
		if _, exists := current[p.PID]; exists {
			return result, errors.New("duplicate process identity")
		}
		current[p.PID] = p
	}
	result.Processes = after
	result.Commands = map[int]string{}
	for _, line := range strings.Split(output.String(), "\n") {
		line = strings.TrimSpace(line)
		pidText, command, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		pid, err := strconv.Atoi(pidText)
		if err != nil {
			continue
		}
		// Commands are meaningful only for a start/parent identity stable across ps.
		p, exists := current[pid]
		if !exists || p.Identity == "" || prior[pid] != p {
			continue
		}
		if _, duplicate := result.Commands[pid]; duplicate {
			return result, errors.New("duplicate process command")
		}
		result.Commands[pid] = strings.TrimSpace(command)
	}
	for _, server := range ParseServerProcesses(output.String()) {
		if server.Session != session {
			continue
		}
		p, exists := current[server.PID]
		if !exists || p.Identity == "" || prior[server.PID] != p {
			return result, errors.New("terminal generation moved during command observation")
		}
		server.Identity = p.Identity
		result.Servers = append(result.Servers, server)
	}
	return result, nil
}
