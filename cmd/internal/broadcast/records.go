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
//
// Reaping and claiming run under one flock on the directory, so two Couches
// starting together can't both judge the same stale lock and both take it.
type runRecords struct {
	dir string
	// runDir is where private directories live; reaping removes only a
	// directory directly inside it. Empty is os.TempDir().
	runDir string
}

type recordData struct {
	Owner      int    `json:"owner_pid"`
	OwnerID    string `json:"owner_identity"`
	Guard      int    `json:"guard_pid,omitempty"`
	GuardID    string `json:"guard_identity,omitempty"`
	Tunnel     int    `json:"tunnel_pid,omitempty"`
	TunnelID   string `json:"tunnel_identity,omitempty"`
	PrivateDir string `json:"private_dir"`
}

// afterRecordRead, when set by a test, runs after reaping has read a record
// and before it acts on it: the seam that lets a test hold that window open.
var afterRecordRead func()

type runRecord struct {
	path string
	data recordData
	in   *runRecords // its directory, whose lock the rewrite takes
}

func identity(pid int) string { return procutil.StrictIdentity(strconv.Itoa(pid)) }

// locked runs f holding an exclusive flock on the records directory.
func (r *runRecords) locked(f func()) error {
	if err := os.MkdirAll(r.dir, 0o700); err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(r.dir, ".lock"), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	f()
	return nil
}

// reapAndClaim clears orphans and then claims key, as one step under the
// lock. A non-empty key is a fixed name (the named-tunnel lock); a key still
// held by a live owner is ErrTunnelBusy.
func (r *runRecords) reapAndClaim(key, dir string) (*runRecord, error) {
	if r == nil {
		return nil, nil
	}
	var rec *runRecord
	var claimErr error
	if err := r.locked(func() {
		r.reapLocked()
		rec, claimErr = r.claimLocked(key, dir)
	}); err != nil {
		return nil, err
	}
	return rec, claimErr
}

func (r *runRecords) claimLocked(key, dir string) (*runRecord, error) {
	if key == "" {
		var b [6]byte
		_, _ = rand.Read(b[:])
		key = "quick-" + strconv.Itoa(os.Getpid()) + "-" + hex.EncodeToString(b[:])
	}
	rec := &runRecord{
		in:   r,
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

// record adds the guard's and cloudflared's identities once they run. An
// unwritable record would leave a crash unreapable, so its error fails the
// open.
func (rec *runRecord) record(guard, tunnel int) error {
	if rec == nil {
		return nil
	}
	rec.data.Guard, rec.data.GuardID = guard, identity(guard)
	if tunnel > 0 {
		rec.data.Tunnel, rec.data.TunnelID = tunnel, identity(tunnel)
	}
	b, err := json.Marshal(rec.data)
	if err != nil {
		return err
	}
	write := func() error {
		tmp := rec.path + ".tmp"
		if err := os.WriteFile(tmp, b, 0o600); err != nil {
			return err
		}
		return os.Rename(tmp, rec.path)
	}
	if rec.in == nil {
		return write()
	}
	// Under the directory lock, so a reaper never sees a .tmp mid-write and
	// can remove any it finds as a crash's leftover.
	var werr error
	if err := rec.in.locked(func() { werr = write() }); err != nil {
		return err
	}
	return werr
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
// runDir is where private directories are made ("" is os.TempDir()).
func ReapOrphans(dir, runDir string) {
	r := &runRecords{dir: dir, runDir: runDir}
	_ = r.locked(r.reapLocked)
}

func (r *runRecords) reapLocked() {
	// A .tmp is only ever mid-write under this lock; one found here was left
	// by a crash between write and rename.
	leftovers, _ := filepath.Glob(filepath.Join(r.dir, "*.json.tmp"))
	for _, tmp := range leftovers {
		_ = os.Remove(tmp)
	}
	paths, _ := filepath.Glob(filepath.Join(r.dir, "*.json"))
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if afterRecordRead != nil {
			afterRecordRead()
		}
		var d recordData
		if json.Unmarshal(b, &d) != nil {
			_ = os.Remove(path)
			continue
		}
		if d.OwnerID == "" {
			// No identity on this platform: a recycled PID could pass for the
			// owner, so kill nothing. A record whose owner PID is gone outright
			// is still cleared, or a named tunnel would stay busy forever.
			if !procutil.Alive(strconv.Itoa(d.Owner)) {
				r.removePrivateDir(d.PrivateDir)
				_ = os.Remove(path)
			}
			continue
		}
		if procutil.Alive(strconv.Itoa(d.Owner)) && identity(d.Owner) == d.OwnerID {
			continue
		}
		killIfOurs(d.Tunnel, d.TunnelID, "cloudflared")
		killIfOurs(d.Guard, d.GuardID, GuardSubcommand)
		r.removePrivateDir(d.PrivateDir)
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

// removePrivateDir removes a recorded private directory only if it is
// plainly one of ours: directly inside the run directory, named with our
// prefix, a real directory (not a symlink) with mode 0700. A record is a file
// on disk; it must not be able to aim RemoveAll anywhere else.
func (r *runRecords) removePrivateDir(dir string) {
	runDir := r.runDir
	if runDir == "" {
		runDir = os.TempDir()
	}
	if dir == "" || !filepath.IsAbs(dir) || filepath.Clean(filepath.Dir(dir)) != filepath.Clean(runDir) ||
		!strings.HasPrefix(filepath.Base(dir), privateDirPrefix) {
		return
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return
	}
	_ = os.RemoveAll(dir)
}
