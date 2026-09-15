// Package diagnosticlog coordinates optional traces without blocking terminal IO.
// The current pathname is stable; immutable generations live next to that path.
package diagnosticlog

import (
	"bytes"
	"context"
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
type Proof func(ctx context.Context, path string, writers []Registration) error
type Options struct {
	// appendWrite injects real partial-write outcomes in durability tests.
	appendWrite func(*os.File, []byte) (int, error)
	appendSync  func(*os.File) error
	// Context bounds optional maintenance; nil preserves ordinary writer behavior.
	Context  context.Context
	Now      func() time.Time
	MaxBytes int64
	Proof    Proof
	Root     string
	Registry func(context.Context, RegistryEntry) error
	Retire   func(context.Context, RegistryEntry) error
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
	Creating  *generation    `json:"creating,omitempty"`
	Appending *appendIntent  `json:"appending,omitempty"`
	Version   int            `json:"version"`
	Path      string         `json:"path"`
	Current   generation     `json:"current"`
	Writers   []Registration `json:"writers"`
	Pending   *generation    `json:"pending,omitempty"`
	Deleting  *generation    `json:"deleting,omitempty"`
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
		if err := recoverPublication(p, options); err != nil {
			return err
		}
		// Registry callbacks are atomic entry writes only and must not acquire
		// root coordination. Publish under this stable lock, so a paused opener
		// republishes after collector retirement before touching content.
		if options.Registry != nil {
			if e := options.Registry(options.context(), RegistryEntry{Version: 1, Path: p, Directory: directory(p), Lock: lockPath(p)}); e != nil {
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
		if e := recoverCreation(p, &s, options); e != nil {
			return e
		}
		if e := recoverAppend(p, &s, options, false); e != nil {
			return e
		}
		if s.Pending == nil && s.Deleting == nil {
			if e := ensureCurrent(p, &s, options); e != nil {
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
		return save(p, s, true, options)
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
		if err := recoverPublication(w.path, w.options); err != nil {
			return err
		}
		s, e := load(w.path)
		if e != nil {
			return e
		}
		if e = validate(s, w.path); e != nil {
			return e
		}
		if e := recoverCreation(w.path, &s, w.options); e != nil {
			return e
		}
		if e := recoverAppend(w.path, &s, w.options, false); e != nil {
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
		if e := ensureCurrent(w.path, &s, w.options); e != nil {
			return e
		}
		for len(b) > 0 {
			chunk := b[:min(len(b), maxAppendBytes)]
			written, e := appendChunk(w.path, &s, chunk, w.options)
			n += written
			if e != nil {
				return e
			}
			b = b[len(chunk):]
		}
		return nil
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
	return w.options.prove(w.path, append([]Registration(nil), s.Writers...))
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
func (o Options) checkContext() error {
	if o.Context != nil {
		return o.Context.Err()
	}
	return nil
}

func lockedOptions(path string, create bool, o Options, fn func() error) error {
	if err := o.checkContext(); err != nil {
		return err
	}
	return locked(path, create, func() error {
		if err := o.checkContext(); err != nil {
			return err
		}
		return fn()
	})
}

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
	if s.Current.Name != "" {
		return errors.New("diagnostic current cannot name a segment")
	}
	if s.Current.Size < 0 {
		return errors.New("invalid diagnostic size")
	}
	for _, r := range s.Writers {
		if r.PID <= 0 || r.Birth == "" {
			return ErrUnknownWriters
		}
	}
	operations := 0
	for _, present := range []bool{s.Pending != nil, s.Deleting != nil, s.Creating != nil, s.Appending != nil} {
		if present {
			operations++
		}
	}
	if operations > 1 {
		return errors.New("conflicting diagnostic operations")
	}
	validGeneration := func(g generation) bool {
		return g.Identity != (identity{}) && !g.Start.IsZero() && !g.LastWrite.IsZero() && !g.ModTime.IsZero() && g.Size >= 0
	}
	if s.Creating != nil && (!validGeneration(*s.Creating) || s.Creating.Size != 0 || s.Creating.Name != "" || s.Current.Identity != (identity{})) {
		return errors.New("invalid creation intent")
	}
	if s.Appending != nil {
		a := s.Appending
		if !validGeneration(a.Before) || a.Before != s.Current || a.At.IsZero() || len(a.Data) == 0 || len(a.Data) > maxAppendBytes {
			return errors.New("invalid append intent")
		}
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
func save(path string, s diskState, durable bool, o Options) error {
	return writeJSON(directory(path), filepath.Join(directory(path), "state.json"), s, durable, o)
}
func writeJSON(stageDir, path string, value any, durable bool, o Options) error {
	return publishJSON(stageDir, path, value, durable, o, nil)
}

func publishJSON(stageDir, path string, value any, durable bool, o Options, afterStage func() error) (err error) {
	if err := o.checkContext(); err != nil {
		return err
	}
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(b) > 1<<20 {
		return errors.New("oversized diagnostic metadata")
	}
	if err := o.checkContext(); err != nil {
		return err
	}
	if err := clearPublication(stageDir, o); err != nil {
		return err
	}
	stage := publicationPath(stageDir)
	if err := o.checkContext(); err != nil {
		return err
	}
	f, err := os.OpenFile(stage, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	defer func() {
		if e := os.Remove(stage); e == nil {
			err = errors.Join(err, syncDir(stageDir))
		} else if !os.IsNotExist(e) {
			err = errors.Join(err, e)
		}
	}()
	if err := o.checkContext(); err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		return err
	}
	if durable {
		if err := o.checkContext(); err != nil {
			return err
		}
		if err := f.Sync(); err != nil {
			return err
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	if afterStage != nil {
		if err := afterStage(); err != nil {
			return err
		}
	}
	if err := o.checkContext(); err != nil {
		return err
	}
	if err := os.Rename(stage, path); err != nil {
		return err
	}
	if durable {
		if stageDir != filepath.Dir(path) {
			return errors.Join(syncDir(stageDir), syncDir(filepath.Dir(path)))
		}
		return syncDir(stageDir)
	}
	return nil
}

// Publishers hold the matching log or registry lock. Only this exact shared
// stage is reclaimed here. Older numeric per-log .pending-* residues retain
// their existing cleanup under the same log lock; registry residues are retained.
func publicationPath(dir string) string { return filepath.Join(dir, ".metadata-publication") }
func clearPublication(dir string, o Options) error {
	if err := o.checkContext(); err != nil {
		return err
	}
	path := publicationPath(dir)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("unsafe diagnostic publication stage")
	}
	if err := o.checkContext(); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	return syncDir(dir)
}
func recoverPublication(path string, o Options) error {
	if err := o.checkContext(); err != nil {
		return err
	}
	return clearPublication(directory(path), o)
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
	if err := o.checkContext(); err != nil {
		return err
	}
	if o.Fault != nil {
		if err := o.Fault(step); err != nil {
			return err
		}
	}
	return o.checkContext()
}
func validSegment(name string) bool {
	if len(name) != len("segment-")+32+len(".log") || name[:8] != "segment-" || name[len(name)-4:] != ".log" {
		return false
	}
	_, e := hex.DecodeString(name[8:40])
	return e == nil
}
func rotate(path string, s *diskState, o Options) error {
	if err := o.checkContext(); err != nil {
		return err
	}
	g := s.Current
	// Hot appends are atomically tracked but best-effort durable. Flush and
	// revalidate their payload before publishing durable rotation authority.
	f, e := openRegular(path, syscall.O_RDONLY)
	if e != nil {
		return e
	}
	st, e := f.Stat()
	if e == nil {
		e = matches(st, g)
	}
	if e == nil {
		e = syncAppendPayload(f, o)
	}
	if e == nil {
		st, e = f.Stat()
		if e == nil {
			e = matches(st, g)
		}
	}
	e = errors.Join(e, f.Close())
	if e != nil {
		return e
	}
	if e := fault(o, "rotation-payload-synced"); e != nil {
		return e
	}
	ready := false
	for range 8 {
		if err := o.checkContext(); err != nil {
			return err
		}
		var token [16]byte
		if _, e := rand.Read(token[:]); e != nil {
			return e
		}
		g.Name = "segment-" + hex.EncodeToString(token[:]) + ".log"
		if e := ensureSegmentDirectory(path, g.Name, o); e != nil {
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
	if err := o.checkContext(); err != nil {
		return err
	}
	if e := save(path, *s, true, o); e != nil {
		return e
	}
	if e := fault(o, "intent"); e != nil {
		return e
	}
	return recoverRotation(path, s, o)
}
func recoverRotation(path string, s *diskState, o Options) error {
	if err := o.checkContext(); err != nil {
		return err
	}
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
		if err := o.checkContext(); err != nil {
			return err
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
	if e = cleanupGenerationTemps(filepath.Dir(dst), o); e != nil {
		return e
	}
	if err := o.checkContext(); err != nil {
		return err
	}
	if e = writeJSON(directory(path), dst+".json", g, true, o); e != nil {
		return e
	}
	s.Current = generation{}
	s.Pending = nil
	if err := o.checkContext(); err != nil {
		return err
	}
	if e = save(path, *s, true, o); e != nil {
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
	if err := options.checkContext(); err != nil {
		return err
	}
	p, e := canonical(path)
	if e != nil {
		return e
	}
	options = normalized(options)
	return lockedOptions(p, false, options, func() error {
		if err := recoverPublication(p, options); err != nil {
			return err
		}
		s, e := load(p)
		if e != nil {
			return e
		}
		if e = validate(s, p); e != nil {
			return e
		}
		if e := recoverCreation(p, &s, options); e != nil {
			return e
		}
		if e := recoverAppend(p, &s, options, true); e != nil {
			return e
		}
		if options.Proof == nil {
			return ErrUnknownWriters
		}
		if e = options.prove(p, s.Writers); e != nil {
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

// prove threads the maintenance lifetime through the single inspection seam.
func (o Options) prove(path string, writers []Registration) error {
	ctx := o.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if o.Proof == nil {
		return ErrUnknownWriters
	}
	err := o.Proof(ctx, path, writers)
	if canceled := ctx.Err(); canceled != nil {
		return canceled
	}
	return err
}

func (o Options) context() context.Context {
	if o.Context != nil {
		return o.Context
	}
	return context.Background()
}
