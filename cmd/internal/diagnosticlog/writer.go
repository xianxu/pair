// Package diagnosticlog coordinates optional traces without blocking terminal IO.
// The current pathname is stable; immutable generations live next to that path.
package diagnosticlog

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

const RetentionPeriod = 7 * 24 * time.Hour
const GenerationPeriod = 24 * time.Hour
const MaxGenerationBytes int64 = 64 << 20

var ErrBusy = errors.New("diagnostic log busy")
var ErrUnknownWriters = errors.New("diagnostic writer inventory incomplete")

type Registration struct {
	PID   int    `json:"pid"`
	Birth string `json:"birth"`
	Root  string `json:"root,omitempty"`
}
type Proof func(path string, writers []Registration) error
type Options struct {
	Now      func() time.Time
	MaxBytes int64
	Proof    Proof
	Root     string
	Registry func(RegistryEntry) error
	Retire   func(RegistryEntry) error
	Fault    func(step string) error
	// SynchronousMaintenance is for explicit batch writers and fault harnesses;
	// terminal emitters leave it false and never inspect processes on append.
	SynchronousMaintenance bool
}
type Writer struct {
	mu                 sync.Mutex
	path               string
	options            Options
	registration       Registration
	closed             bool
	maintenance        sync.WaitGroup
	maintenanceRunning bool
	lastMaintenance    time.Time
}
type identity struct {
	Device uint64 `json:"device"`
	Inode  uint64 `json:"inode"`
}
type generation struct {
	Name      string    `json:"name"`
	Identity  identity  `json:"identity"`
	Start     time.Time `json:"start"`
	LastWrite time.Time `json:"last_write"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"mod_time"`
}
type diskState struct {
	Version  int            `json:"version"`
	Path     string         `json:"path"`
	Current  generation     `json:"current"`
	Writers  []Registration `json:"writers"`
	Pending  *generation    `json:"pending,omitempty"`
	Deleting *generation    `json:"deleting,omitempty"`
}

func normalized(opts Options) Options {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = MaxGenerationBytes
	}
	return opts
}
func canonical(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty diagnostic path")
	}
	p, e := filepath.Abs(path)
	if e != nil {
		return "", e
	}
	dir, e := filepath.EvalSymlinks(filepath.Dir(p))
	if e != nil {
		return "", e
	}
	p = filepath.Join(dir, filepath.Base(p))
	if st, e := os.Lstat(p); e == nil && !st.Mode().IsRegular() {
		return "", errors.New("diagnostic path is not regular")
	} else if e != nil && !os.IsNotExist(e) {
		return "", e
	}
	return p, nil
}
func Open(path string, options Options) (*Writer, error) {
	p, e := canonical(path)
	if e != nil {
		return nil, e
	}
	options = normalized(options)
	reg := Registration{PID: os.Getpid(), Birth: procutil.StrictIdentity(strconv.Itoa(os.Getpid())), Root: options.Root}
	if reg.Birth == "" {
		return nil, ErrUnknownWriters
	}

	w := &Writer{path: p, options: options, registration: reg}
	e = locked(p, true, func() error {
		// Registry callbacks are atomic entry writes only and must not acquire
		// root coordination. Publish under this stable lock, so a paused opener
		// republishes after collector retirement before touching content.
		if options.Registry != nil {
			if e := options.Registry(RegistryEntry{Version: 1, Path: p, Directory: directory(p), Lock: lockPath(p)}); e != nil {
				return e
			}
		}
		s, e := load(p)
		if os.IsNotExist(e) {
			s = diskState{Version: 1, Path: p}
			e = nil
		}
		if e != nil {
			return e
		}
		if e = validate(s, p); e != nil {
			return e
		}
		if s.Pending == nil && s.Deleting == nil {
			f, e := openRegular(p, syscall.O_CREAT|syscall.O_APPEND|syscall.O_WRONLY)
			if e != nil {
				return e
			}
			st, e := f.Stat()
			f.Close()
			if e != nil {
				return e
			}
			if s.Current.Identity == (identity{}) {
				s.Current = generation{Identity: fileIdentity(st), Start: options.Now().UTC(), LastWrite: st.ModTime(), Size: st.Size(), ModTime: st.ModTime()}
			} else if e = matches(st, s.Current); e != nil {
				return e
			}
		}
		found := false
		out := s.Writers[:0]
		for _, r := range s.Writers {
			if r == reg {
				found = true
			}
			if r == reg || !definitelyDead(r) {
				out = append(out, r)
			}
		}
		s.Writers = out
		if !found {
			s.Writers = append(s.Writers, reg)
		}
		if len(s.Writers) > 1024 {
			return errors.New("too many diagnostic writers")
		}
		return save(p, s, true)
	})
	if e != nil {
		return nil, e
	}
	return w, nil
}
func (w *Writer) Close() error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	w.closed = true
	w.mu.Unlock()
	w.maintenance.Wait()
	// Process registrations survive individual handles and syscall.Exec.
	return nil
}
func (w *Writer) Write(b []byte) (int, error) {
	if !w.mu.TryLock() {
		return 0, ErrBusy
	}
	defer w.mu.Unlock()
	if w.closed {
		return 0, os.ErrClosed
	}
	if len(b) == 0 {
		return 0, nil
	}
	n := 0
	maintenanceDue := false
	e := locked(w.path, false, func() error {
		s, e := load(w.path)
		if e != nil {
			return e
		}
		if e = validate(s, w.path); e != nil {
			return e
		}
		if s.Deleting != nil || s.Pending != nil {
			if !w.options.SynchronousMaintenance {
				maintenanceDue = true
				return ErrBusy
			}
		}
		if s.Deleting != nil {
			if e = w.prove(s); e != nil {
				return e
			}
			if e = recoverDeletion(w.path, &s, w.options); e != nil {
				return e
			}
		}
		if s.Pending != nil {
			if e = w.prove(s); e != nil {
				return e
			}
			if e = recoverRotation(w.path, &s, w.options); e != nil {
				return e
			}
		}
		st, e := regular(w.path)
		exists := e == nil
		if e != nil && !os.IsNotExist(e) {
			return e
		}
		if exists && s.Current.Identity != (identity{}) {
			if e = matches(st, s.Current); e != nil {
				return e
			}
		}
		now := w.options.Now().UTC()
		if !s.Current.Start.IsZero() && exists && (st.Size() >= w.options.MaxBytes || !now.Before(s.Current.Start.Add(GenerationPeriod))) {
			maintenanceDue = true
			if w.options.SynchronousMaintenance && w.prove(s) == nil {
				if e = rotate(w.path, &s, w.options); e != nil {
					return e
				}
				exists = false
			}
		}
		f, e := openRegular(w.path, syscall.O_CREAT|syscall.O_APPEND|syscall.O_WRONLY)
		if e != nil {
			return e
		}
		defer f.Close()
		if !exists {
			st, e = f.Stat()
			if e != nil {
				return e
			}
			s.Current = generation{Identity: fileIdentity(st), Start: now, LastWrite: now}
		}
		if s.Current.Start.IsZero() {
			st, e = f.Stat()
			if e != nil {
				return e
			}
			s.Current = generation{Identity: fileIdentity(st), Start: now, LastWrite: st.ModTime(), Size: st.Size()}
		}
		n, e = f.Write(b)
		if e != nil {
			return e
		}
		st, e = f.Stat()
		if e != nil {
			return e
		}
		s.Current.Size = st.Size()
		s.Current.ModTime = st.ModTime()
		s.Current.LastWrite = now
		// LastWrite is explicit so an injected clock and coarse filesystem clocks
		// cannot make a young managed generation appear old.
		return save(w.path, s, false)
	})
	if maintenanceDue && !w.options.SynchronousMaintenance {
		w.scheduleMaintenance()
	}
	return n, e
}
func (w *Writer) prove(s diskState) error {
	if w.options.Proof == nil {
		return ErrUnknownWriters
	}
	return w.options.Proof(w.path, append([]Registration(nil), s.Writers...))
}
func definitelyDead(r Registration) bool {
	if r.PID <= 0 || r.Birth == "" {
		return false
	}
	if e := syscall.Kill(r.PID, 0); errors.Is(e, syscall.ESRCH) {
		return true
	}
	birth := procutil.StrictIdentity(strconv.Itoa(r.PID))
	return birth != "" && birth != r.Birth
}
func directory(path string) string { return path + ".pair-diagnostics" }
func lockPath(path string) string  { return path + ".pair-diagnostics.lock" }
func locked(path string, create bool, fn func() error) error {
	flags := syscall.O_RDWR
	if create {
		flags |= syscall.O_CREAT
	}
	f, e := openRegular(lockPath(path), flags)
	if e != nil {
		return e
	}
	defer f.Close()
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		if errors.Is(e, syscall.EWOULDBLOCK) {
			return ErrBusy
		}
		return e
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	if create {
		if e = os.Mkdir(directory(path), 0700); e != nil && !os.IsExist(e) {
			return e
		} else if e == nil {
			if e = syncDir(filepath.Dir(path)); e != nil {
				return e
			}
		}
	}
	st, e := os.Lstat(directory(path))
	if !create && os.IsNotExist(e) {
		return fn()
	}
	if e != nil {
		return e
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return errors.New("invalid diagnostic directory")
	}
	return fn()
}
func openRegular(path string, flags int) (*os.File, error) {
	fd, e := syscall.Open(path, flags|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(fd), path)
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() {
		f.Close()
		return nil, errors.New("diagnostic file is not regular")
	}
	return f, nil
}
func regular(path string) (os.FileInfo, error) {
	st, e := os.Lstat(path)
	if e == nil && !st.Mode().IsRegular() {
		return nil, errors.New("diagnostic file is not regular")
	}
	return st, e
}
func fileIdentity(st os.FileInfo) identity {
	s := st.Sys().(*syscall.Stat_t)
	return identity{uint64(s.Dev), uint64(s.Ino)}
}
func matches(st os.FileInfo, g generation) error {
	if fileIdentity(st) != g.Identity || st.Size() != g.Size || !st.ModTime().Equal(g.ModTime) {
		return errors.New("diagnostic generation changed outside protocol")
	}
	return nil
}
func load(path string) (diskState, error) {
	var s diskState
	e := readJSON(filepath.Join(directory(path), "state.json"), &s)
	return s, e
}
func readJSON(path string, dst any) error {
	f, e := openRegular(path, syscall.O_RDONLY)
	if e != nil {
		return e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if e != nil {
		return e
	}
	if len(b) > 1<<20 {
		return errors.New("oversized diagnostic metadata")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(dst); e != nil {
		return e
	}
	var extra any
	if e = d.Decode(&extra); e != io.EOF {
		return errors.New("trailing diagnostic metadata")
	}
	return nil
}
func validate(s diskState, path string) error {
	if s.Version != 1 || s.Path != path {
		return errors.New("invalid diagnostic identity")
	}
	if s.Current.Size < 0 {
		return errors.New("invalid diagnostic size")
	}
	for _, r := range s.Writers {
		if r.PID <= 0 || r.Birth == "" {
			return ErrUnknownWriters
		}
	}
	if s.Pending != nil && s.Deleting != nil {
		return errors.New("conflicting diagnostic operations")
	}
	validGeneration := func(g generation) bool {
		return g.Identity != (identity{}) && !g.Start.IsZero() && !g.LastWrite.IsZero() && !g.ModTime.IsZero() && g.Size >= 0
	}
	if s.Current.Identity != (identity{}) && !validGeneration(s.Current) {
		return errors.New("invalid current diagnostic generation")
	}
	if s.Pending != nil && (!validGeneration(*s.Pending) || !validSegment(s.Pending.Name)) {
		return errors.New("invalid rotation intent")
	}
	if s.Deleting != nil && (!validGeneration(*s.Deleting) || (s.Deleting.Name != "" && !validSegment(s.Deleting.Name))) {
		return errors.New("invalid deletion intent")
	}

	return nil
}
func save(path string, s diskState, durable bool) error {
	return writeJSON(filepath.Join(directory(path), "state.json"), s, durable)
}
func writeJSON(path string, value any, durable bool) error {
	b, e := json.Marshal(value)
	if e != nil {
		return e
	}
	if len(b) > 1<<20 {
		return errors.New("oversized diagnostic metadata")
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".pending-")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, e = f.Write(b); e == nil && durable {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	if e = os.Rename(tmp, path); e != nil {
		return e
	}
	if durable {
		return syncDir(filepath.Dir(path))
	}
	return nil
}
func syncDir(path string) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	return f.Sync()
}
func fault(o Options, step string) error {
	if o.Fault != nil {
		return o.Fault(step)
	}
	return nil
}
func validSegment(name string) bool {
	if len(name) != len("segment-")+32+len(".log") || name[:8] != "segment-" || name[len(name)-4:] != ".log" {
		return false
	}
	_, e := hex.DecodeString(name[8:40])
	return e == nil
}
func rotate(path string, s *diskState, o Options) error {
	g := s.Current
	ready := false
	for range 8 {
		var token [16]byte
		if _, e := rand.Read(token[:]); e != nil {
			return e
		}
		g.Name = "segment-" + hex.EncodeToString(token[:]) + ".log"
		if e := ensureSegmentDirectory(path, g.Name); e != nil {
			return e
		}
		names, e := directoryNames(filepath.Dir(segmentPath(path, g.Name)), 2*leafGenerations+1)
		if e != nil {
			return e
		}
		if len(names) < 2*leafGenerations {
			ready = true
			break
		}
	}
	if !ready {
		return errors.New("diagnostic radix allocation busy")
	}
	s.Pending = &g
	if e := save(path, *s, true); e != nil {
		return e
	}
	if e := fault(o, "intent"); e != nil {
		return e
	}
	return recoverRotation(path, s, o)
}
func recoverRotation(path string, s *diskState, o Options) error {
	g := *s.Pending
	dst := segmentPath(path, g.Name)
	st, e := regular(dst)
	if os.IsNotExist(e) {
		st, e = regular(path)
		if e != nil {
			return e
		}
		if e = matches(st, g); e != nil {
			return e
		}
		if e = os.Rename(path, dst); e != nil {
			return e
		}
		if e = syncDir(filepath.Dir(path)); e != nil {
			return e
		}
		if e = syncDir(filepath.Dir(dst)); e != nil {
			return e
		}
	} else if e != nil {
		return e
	} else if e = matches(st, g); e != nil {
		return e
	}
	if _, e := os.Lstat(path); !os.IsNotExist(e) {
		return errors.New("diagnostic current replaced during rotation")
	}

	if e = fault(o, "rename"); e != nil {
		return e
	}
	if e = cleanupGenerationTemps(filepath.Dir(dst)); e != nil {
		return e
	}
	if e = writeJSON(dst+".json", g, true); e != nil {
		return e
	}
	s.Current = generation{}
	s.Pending = nil
	if e = save(path, *s, true); e != nil {
		return e
	}
	return fault(o, "state")
}

// Append is the small internal CLI seam used by shell and Lua emitters. Caller
// supplies an already encoded record; this package does not reinterpret payloads.
func Append(path string, record []byte, options Options) error {
	w, e := Open(path, options)
	if e != nil {
		return e
	}
	defer w.Close()
	_, e = w.Write(record)
	return e
}

var _ io.WriteCloser = (*Writer)(nil)

// Called with w.mu held. Failed proof gets a cooldown, so a legacy writer
// cannot cause one process scan per terminal chunk.
func (w *Writer) scheduleMaintenance() {
	now := w.options.Now()
	if w.maintenanceRunning || (!w.lastMaintenance.IsZero() && now.Before(w.lastMaintenance.Add(time.Minute))) {
		return
	}
	w.maintenanceRunning = true
	w.lastMaintenance = now
	w.maintenance.Go(func() { _ = Maintain(w.path, w.options); w.mu.Lock(); w.maintenanceRunning = false; w.mu.Unlock() })
}

// Maintain is scheduled work, never a terminal callback. It performs one
// pending recovery or generation rotation, with nonblocking coordination.
func Maintain(path string, options Options) error {
	p, e := canonical(path)
	if e != nil {
		return e
	}
	options = normalized(options)
	return locked(p, false, func() error {
		s, e := load(p)
		if e != nil {
			return e
		}
		if e = validate(s, p); e != nil {
			return e
		}
		if options.Proof == nil {
			return ErrUnknownWriters
		}
		if e = options.Proof(p, s.Writers); e != nil {
			return e
		}
		if s.Deleting != nil {
			if e = recoverDeletion(p, &s, options); e != nil {
				return e
			}
		}
		if s.Pending != nil {
			return recoverRotation(p, &s, options)
		}
		if s.Current.Identity == (identity{}) {
			return nil
		}
		st, e := regular(p)
		if e != nil {
			return e
		}
		if e = matches(st, s.Current); e != nil {
			return e
		}
		if !options.Now().Before(s.Current.Start.Add(GenerationPeriod)) || st.Size() >= options.MaxBytes {
			return rotate(p, &s, options)
		}
		return nil
	})
}
