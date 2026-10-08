package broadcast

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// runRecords is a directory of run records, one JSON file per running
// tunnel. A record names the owner Couch, the guard and cloudflared, each
// with its kernel incarnation identity (procutil.StrictIdentity), so a
// recycled PID is never mistaken for them. For a named tunnel the record's
// fixed name is also the lock that keeps a second Couch off that tunnel.
// A nil *runRecords disables all of it.
type runRecords struct{ dir string }

type recordData struct {
	Owner      int    `json:"owner_pid"`
	OwnerID    string `json:"owner_identity"`
	Guard      int    `json:"guard_pid,omitempty"`
	GuardID    string `json:"guard_identity,omitempty"`
	Tunnel     int    `json:"tunnel_pid,omitempty"`
	TunnelID   string `json:"tunnel_identity,omitempty"`
	PrivateDir string `json:"private_dir"`
}

type runRecord struct {
	path string
	data recordData
}

func identity(pid int) string { return procutil.StrictIdentity(strconv.Itoa(pid)) }

// claim creates this process's record for a tunnel serving from dir. A
// non-empty key is a fixed name (the named-tunnel lock); a key already held
// by a live owner is ErrTunnelBusy.
func (r *runRecords) claim(key, dir string) (*runRecord, error) {
	if r == nil {
		return nil, nil
	}
	if err := os.MkdirAll(r.dir, 0o700); err != nil {
		return nil, err
	}
	if key == "" {
		var b [6]byte
		_, _ = rand.Read(b[:])
		key = "quick-" + strconv.Itoa(os.Getpid()) + "-" + hex.EncodeToString(b[:])
	}
	rec := &runRecord{
		path: filepath.Join(r.dir, key+".json"),
		data: recordData{Owner: os.Getpid(), OwnerID: identity(os.Getpid()), PrivateDir: dir},
	}
	f, err := os.OpenFile(rec.path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return nil, ErrTunnelBusy
	}
	if err != nil {
		return nil, err
	}
	err = json.NewEncoder(f).Encode(rec.data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(rec.path)
		return nil, err
	}
	return rec, nil
}

// record adds the guard's and cloudflared's identities once they run.
func (rec *runRecord) record(guard, tunnel int) {
	if rec == nil {
		return
	}
	rec.data.Guard, rec.data.GuardID = guard, identity(guard)
	if tunnel > 0 {
		rec.data.Tunnel, rec.data.TunnelID = tunnel, identity(tunnel)
	}
	b, err := json.Marshal(rec.data)
	if err != nil {
		return
	}
	tmp := rec.path + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, rec.path)
	}
}

func (rec *runRecord) release() {
	if rec != nil {
		_ = os.Remove(rec.path)
	}
}

// ReapOrphans clears tunnels left behind by a Couch that died without
// stopping them: its guard normally does that, so this is the backstop for
// a guard that was itself killed. A record is reaped only when its owner is
// gone (dead, or its PID now belongs to another incarnation). Processes are
// killed only if their identity matches the record and their command is what
// it should be; the private directory is removed only if it is one of ours.
func ReapOrphans(dir string) {
	(&runRecords{dir: dir}).reap()
}

func (r *runRecords) reap() {
	if r == nil {
		return
	}
	paths, _ := filepath.Glob(filepath.Join(r.dir, "*.json"))
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var d recordData
		if json.Unmarshal(b, &d) != nil {
			_ = os.Remove(path)
			continue
		}
		if d.OwnerID == "" {
			// No identity on this platform: can't tell a recycled PID from
			// the owner, so destroy nothing.
			continue
		}
		if procutil.Alive(strconv.Itoa(d.Owner)) && identity(d.Owner) == d.OwnerID {
			continue
		}
		killIfOurs(d.Tunnel, d.TunnelID, "cloudflared")
		killIfOurs(d.Guard, d.GuardID, GuardSubcommand)
		if strings.HasPrefix(filepath.Base(d.PrivateDir), privateDirPrefix) {
			_ = os.RemoveAll(d.PrivateDir)
		}
		_ = os.Remove(path)
	}
}

// killIfOurs kills pid's process group if pid is still the recorded
// incarnation running the expected command.
func killIfOurs(pid int, id, command string) {
	if pid <= 0 || id == "" {
		return
	}
	p := strconv.Itoa(pid)
	if !procutil.Alive(p) || identity(pid) != id || !strings.Contains(procutil.Command(p), command) {
		return
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	_ = syscall.Kill(pid, syscall.SIGKILL)
}
