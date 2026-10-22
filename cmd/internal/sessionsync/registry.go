package sessionsync

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/durablefile"
)

// Registration is the public view. List's Owner is the persisted fallback;
// single-document commands resolve the effective Owner from current content.
type Registration struct {
	ID, File, Owner, RegisteredOwner string
	RegisteredAt                     time.Time
}

// DocumentInfo is a fresh registered document's routing and operator-lock facts.
// Inspecting these facts does not acknowledge any document content or delivery.
type DocumentInfo struct {
	ID, File, Owner string
	Holder          Holder
	SessionSync     bool
}

// InspectRegistered reads current facts under the document's transaction lock.
// A legacy session-sync file is insufficient authority without a registration.
func (s *Sync) InspectRegistered(path string) (DocumentInfo, error) {
	return withDocumentMode(s, path, registeredOnly, func(f *file) (DocumentInfo, error) {
		if !ValidSlot(f.owner) {
			return DocumentInfo{}, refuse("document owner %q is invalid", f.owner)
		}
		return DocumentInfo{ID: f.registration.ID, File: f.registration.File, Owner: f.owner, Holder: f.holder, SessionSync: IsSessionSync(f.content)}, nil
	})
}

const registeredStateMaxBytes = 1 << 20

// Preflight every prospective registered companion before publishing authority
// or mutating the shared document. Producers and readers share these limits.
func validateRegisteredMetadata(shadow string, st State, paths []PathInterval) error {
	// Reserve the complete legacy handoff representation, not only central bytes.
	st.SubmitPaths = paths
	if len(shadow) > documentMaxBytes {
		return registryLimit()
	}
	raw, err := encodeState(st)
	if err != nil {
		return err
	}
	if len(raw) > registeredStateMaxBytes {
		return registryLimit()
	}
	return nil
}

type RegistryResult struct {
	Registration *Registration
	Changed      bool
}

// registryStore is portable, namespace-local storage. The publisher seam lets
// tests retain real filesystem effects while failing individual durability steps.
type registryStore struct {
	Dir     string
	Publish func(string, []byte) error
	SyncDir func(string) error
	// TryLock is the nonblocking advisory-lock IO boundary.
	TryLock       func(int) error
	OperatorLocks operatorLockFiles
}

type registryTxn struct {
	store *registryStore
	state RegistryState
}

func (s *Sync) registryStore() *registryStore {
	if s.Registry != nil {
		return s.Registry
	}
	return &registryStore{Dir: filepath.Join(s.Ledger.Dir, "documents")}
}

func (s *Sync) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// canonicalDocument also resolves a missing leaf through existing symlinked
// parents, so an editor's post-rename OLD still names the stored identity.
func canonicalDocument(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if strings.ContainsAny(abs, "\x00\r\n") {
		return "", refuse("document path contains a control character")
	}
	var tail []string
	p := abs
	for {
		real, err := filepath.EvalSymlinks(p)
		if err == nil {
			for i := len(tail) - 1; i >= 0; i-- {
				real = filepath.Join(real, tail[i])
			}
			return real, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(p)
		if parent == p {
			return "", err
		}
		tail = append(tail, filepath.Base(p))
		p = parent
	}
}

// readRegular cannot block on a FIFO and checks both type and byte limits.
func readRegular(path string, max int64) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	if info.Size() > max {
		return nil, registryRefusal("registry-limit", fmt.Sprintf("%s exceeds its %d-byte limit", path, max))
	}
	raw, err := io.ReadAll(io.LimitReader(f, max+1))
	if err == nil && int64(len(raw)) > max {
		err = registryRefusal("registry-limit", fmt.Sprintf("%s exceeds its %d-byte limit", path, max))
	}
	return raw, err
}

func withRegistry[T any](s *Sync, fn func(*registryTxn) (T, error)) (T, error) {
	var zero T
	store := s.registryStore()
	if err := durablefile.EnsureDirectory(store.Dir); err != nil {
		return zero, registryAccessFailure(err)
	}
	lock, err := os.OpenFile(filepath.Join(store.Dir, "registry.flock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return zero, registryAccessFailure(err)
	}
	defer lock.Close()
	deadline := time.Now().Add(2 * time.Second)
	tryLock := store.TryLock
	if tryLock == nil {
		tryLock = func(fd int) error { return syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB) }
	}
	for {
		err = tryLock(int(lock.Fd()))
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			return zero, registryAccessFailure(err)
		}
		if !time.Now().Before(deadline) {
			return zero, registryRefusal("registry-busy", "agentic registry is busy; retry this command after the current operation finishes")
		}
		time.Sleep(5 * time.Millisecond)
	}
	state := RegistryState{Version: 1}
	raw, err := readRegular(filepath.Join(store.Dir, "registry.json"), registryMaxBytes)
	if err == nil {
		state, err = decodeRegistry(raw)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return zero, registryAccessFailure(err)
	}
	return fn(&registryTxn{store: store, state: state})
}

func (t *registryTxn) confirm() error {
	fn := t.store.SyncDir
	if fn == nil {
		fn = durablefile.SyncDirectory
	}
	if err := fn(t.store.Dir); err != nil {
		return registryRefusal("registry-uncertain", fmt.Sprintf("registry durability is unconfirmed: %v; retry the identical command and arguments", err))
	}
	return nil
}

func (t *registryTxn) publish(path string, raw []byte) error {
	fn := t.store.Publish
	if fn == nil {
		fn = writeAtomic
	}
	return fn(path, raw)
}

func (t *registryTxn) save(next RegistryState) error {
	raw, err := encodeRegistry(next)
	if err != nil {
		return err
	}
	if err := t.publish(filepath.Join(t.store.Dir, "registry.json"), raw); err != nil {
		return registryRefusal("registry-uncertain", fmt.Sprintf("registry publication is unconfirmed: %v; retry the identical command and arguments", err))
	}
	t.state = next
	return t.confirm()
}

func (t *registryTxn) metadata(d DocumentRegistration) Paths {
	p := PathsOf(d.File)
	root := filepath.Join(t.store.Dir, "state", d.ID)
	p.Shadow = filepath.Join(root, "shadow")
	p.State = filepath.Join(root, "state.json")
	p.Round = filepath.Join(root, "round.json")
	return p
}

func view(d DocumentRegistration, content string, effective bool) *Registration {
	owner := d.RegisteredOwner
	if effective {
		owner = EffectiveOwner(content, owner)
	}
	return &Registration{ID: d.ID, File: d.File, Owner: owner, RegisteredOwner: d.RegisteredOwner, RegisteredAt: d.RegisteredAt}
}

func (t *registryTxn) find(path string) *DocumentRegistration {
	for i := range t.state.Documents {
		if t.state.Documents[i].File == path {
			return &t.state.Documents[i]
		}
	}
	return nil
}

// cleanup runs only while all registered operations are excluded. Never unlink
// the namespace flock inode; forgotten and unpublished preparation dirs are safe.
func (t *registryTxn) cleanup() error {
	root := filepath.Join(t.store.Dir, "state")
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	live := map[string]bool{}
	for _, d := range t.state.Documents {
		live[d.ID] = true
	}
	for _, e := range entries {
		if !live[e.Name()] {
			if err := os.RemoveAll(filepath.Join(root, e.Name())); err != nil {
				return err
			}
		}
	}
	return durablefile.SyncDirectory(root)
}

func (s *Sync) Register(path, owner string) (RegistryResult, error) {
	return withRegistry(s, func(t *registryTxn) (RegistryResult, error) {
		documentPath, err := canonicalDocument(path)
		if err != nil {
			return RegistryResult{}, err
		}
		content, err := readRegular(documentPath, documentMaxBytes)
		if err != nil {
			return RegistryResult{}, err
		}
		if !ValidSlot(owner) {
			return RegistryResult{}, refuse("owner %q is not an exact Couch slot address", owner)
		}
		// Explicit routing metadata is authoritative, including registration retries.
		if explicit := frontmatter(string(content), "couch-owner"); explicit != "" {
			owner = explicit
		}
		if !ValidSlot(EffectiveOwner(string(content), owner)) {
			return RegistryResult{}, refuse("document owner is not an exact Couch slot address")
		}
		spelling, err := filepath.Abs(path)
		if err != nil {
			return RegistryResult{}, err
		}
		if existing := t.find(documentPath); existing != nil {
			requested := owner
			if frontmatter(string(content), "couch-owner") != "" {
				requested = existing.RegisteredOwner
			}
			next, change, err := t.state.transition(RegistryEvent{Kind: "register", File: documentPath, Owner: requested, Spelling: spelling})
			if err != nil {
				return RegistryResult{}, err
			}
			if change.Changed {
				if err := t.validateMetadata(*change.Registration); err != nil {
					return RegistryResult{}, err
				}
				err = t.save(next)
			} else {
				err = t.confirm()
			}
			if err != nil {
				return RegistryResult{}, err
			}
			return RegistryResult{Registration: view(*change.Registration, string(content), true), Changed: change.Changed}, nil
		}
		if err := t.cleanup(); err != nil {
			return RegistryResult{}, err
		}
		var id [16]byte
		if _, err := rand.Read(id[:]); err != nil {
			return RegistryResult{}, err
		}
		at := s.now()
		event := RegistryEvent{Kind: "register", File: documentPath, Owner: owner, ID: hex.EncodeToString(id[:]), At: at, Spelling: spelling}
		st := State{Since: at}
		var shadow string
		var hasShadow bool
		if IsSessionSync(string(content)) {
			for _, companion := range []struct {
				path  string
				limit int64
			}{{PathsOf(path).State, registeredStateMaxBytes}, {PathsOf(path).Shadow, documentMaxBytes}} {
				if _, err := readRegular(companion.path, companion.limit); err != nil && !errors.Is(err, os.ErrNotExist) {
					return RegistryResult{}, err
				}
			}
			_, err := locked(path, func(f *file) (bool, error) {
				st = f.state
				shadow = f.shadow
				hasShadow = f.hasShadow
				event.InitialPaths = f.state.SubmitPaths
				return true, nil
			})
			if err != nil {
				return RegistryResult{}, err
			}
			if len(event.InitialPaths) == 0 {
				event.InitialPaths = []PathInterval{{File: documentPath, Spelling: spelling, From: st.Since}}
			}
		}
		next, change, err := t.state.transition(event)
		if err != nil {
			return RegistryResult{}, err
		}
		p := t.metadata(*change.Registration)
		st.SubmitPaths = nil // registry history is authoritative until a legacy handoff
		if err := validateRegisteredMetadata(shadow, st, change.Registration.Paths); err != nil {
			return RegistryResult{}, err
		}
		if err := saveStateUsing(p, st, t.publish); err != nil {
			return RegistryResult{}, err
		}
		if hasShadow {
			if err := t.publish(p.Shadow, []byte(shadow)); err != nil {
				return RegistryResult{}, err
			}
		}
		// A legacy round comes along; one that is malformed is left behind.
		if raw, err := readRound(PathsOf(path).Round); err == nil && raw != nil {
			if err := t.publish(p.Round, raw); err != nil {
				return RegistryResult{}, err
			}
		}
		if err := t.save(next); err != nil {
			return RegistryResult{}, err
		}
		return RegistryResult{Registration: view(*change.Registration, string(content), true), Changed: true}, nil
	})
}

func (s *Sync) Lookup(path string) (*Registration, error) {
	return withRegistry(s, func(t *registryTxn) (*Registration, error) {
		file, err := canonicalDocument(path)
		if err != nil {
			return nil, err
		}
		d := t.find(file)
		if d == nil {
			return nil, nil
		}
		raw, err := readRegular(file, documentMaxBytes)
		if err != nil {
			return nil, err
		}
		v := view(*d, string(raw), true)
		if !ValidSlot(v.Owner) {
			return nil, refuse("document owner %q is invalid", v.Owner)
		}
		return v, nil
	})
}

func (s *Sync) List() ([]Registration, error) {
	return withRegistry(s, func(t *registryTxn) ([]Registration, error) {
		out := make([]Registration, 0, len(t.state.Documents))
		for _, d := range t.state.Documents {
			out = append(out, *view(d, "", false))
		}
		return out, nil
	})
}

func (s *Sync) Rename(oldPath, newPath string) (RegistryResult, error) {
	return withRegistry(s, func(t *registryTxn) (RegistryResult, error) {
		old, err := canonicalDocument(oldPath)
		if err != nil {
			return RegistryResult{}, err
		}
		if t.find(old) == nil {
			if _, statErr := os.Lstat(oldPath); errors.Is(statErr, os.ErrNotExist) {
				spelling, absErr := filepath.Abs(oldPath)
				if absErr != nil {
					return RegistryResult{}, absErr
				}
				for _, d := range t.state.Documents {
					known := false
					current := d.Paths[len(d.Paths)-1]
					for _, candidate := range append([]string{current.Spelling}, current.Spellings...) {
						if candidate == spelling {
							known = true
							break
						}
					}
					if known {
						old = d.File
						break
					}
				}
			}
		}
		file, err := canonicalDocument(newPath)
		if err != nil {
			return RegistryResult{}, err
		}
		if t.find(old) == nil {
			if destination := t.find(file); destination != nil && len(destination.Paths) > 1 {
				previous := destination.Paths[len(destination.Paths)-2]
				for _, spelling := range append([]string{previous.Spelling}, previous.Spellings...) {
					if spelling == "" {
						continue
					}
					resolved, resolveErr := canonicalDocument(spelling)
					if resolveErr == nil && resolved == old {
						old = spelling
						break
					}
				}
			}
		}
		raw, err := readRegular(file, documentMaxBytes)
		if err != nil {
			return RegistryResult{}, err
		}
		spelling, err := filepath.Abs(newPath)
		if err != nil {
			return RegistryResult{}, err
		}
		next, change, err := t.state.transition(RegistryEvent{Kind: "rename", Old: old, File: file, Spelling: spelling, At: s.now()})
		if err != nil {
			return RegistryResult{}, err
		}
		if change.Changed {
			if err := t.validateMetadata(*change.Registration); err != nil {
				return RegistryResult{}, err
			}
			err = t.save(next)
		} else {
			err = t.confirm()
		}
		if err != nil {
			return RegistryResult{}, err
		}
		return RegistryResult{Registration: view(*change.Registration, string(raw), true), Changed: change.Changed}, nil
	})
}

func (s *Sync) Forget(path string) (RegistryResult, error) {
	return withRegistry(s, func(t *registryTxn) (RegistryResult, error) {
		documentPath, err := canonicalDocument(path)
		if err != nil {
			return RegistryResult{}, err
		}
		d := t.find(documentPath)
		if d != nil {
			raw, err := readRegular(documentPath, documentMaxBytes)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return RegistryResult{}, err
			}
			if err == nil && IsSessionSync(string(raw)) {
				spelling, err := filepath.Abs(path)
				if err != nil {
					return RegistryResult{}, err
				}
				seen := map[string]bool{}
				for _, legacyPath := range append(documentSpellings(*d), spelling) {
					if legacyPath == "" || seen[legacyPath] {
						continue
					}
					seen[legacyPath] = true
					resolved, resolveErr := filepath.EvalSymlinks(legacyPath)
					if errors.Is(resolveErr, os.ErrNotExist) {
						continue
					}
					if resolveErr != nil {
						return RegistryResult{}, resolveErr
					}
					if resolved != documentPath {
						continue
					} // never export onto a reused old name
					if err := t.exportLegacy(*d, legacyPath); err != nil {
						return RegistryResult{}, err
					}
				}
			}
		}
		next, change, err := t.state.transition(RegistryEvent{Kind: "forget", File: documentPath})
		if err != nil {
			return RegistryResult{}, err
		}
		if change.Changed {
			err = t.save(next)
		} else {
			err = t.confirm()
		}
		if err != nil {
			return RegistryResult{}, err
		}
		if err := t.cleanup(); err != nil {
			return RegistryResult{}, registryRefusal("registry-uncertain", fmt.Sprintf("registration forgotten but metadata cleanup is incomplete: %v; retry the identical forget", err))
		}
		return RegistryResult{Changed: change.Changed}, nil
	})
}

// loadMetadata applies the registered reader envelope in one place.
func (t *registryTxn) loadMetadata(d DocumentRegistration) (State, []byte, bool, error) {
	p := t.metadata(d)
	if _, err := readRegular(p.State, registeredStateMaxBytes); err != nil {
		return State{}, nil, false, fmt.Errorf("registered document state unavailable: %w", err)
	}
	st, err := LoadState(p)
	if err != nil {
		return State{}, nil, false, err
	}
	shadow, err := readRegular(p.Shadow, documentMaxBytes)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return State{}, nil, false, err
	}
	return st, shadow, err == nil, nil
}

func (t *registryTxn) validateMetadata(d DocumentRegistration) error {
	st, shadow, _, err := t.loadMetadata(d)
	if err != nil {
		return err
	}
	return validateRegisteredMetadata(string(shadow), st, d.Paths)
}

type documentMode uint8

const (
	legacyCompatible documentMode = iota
	registeredOnly
)

func withDocument[T any](s *Sync, path string, fn func(*file) (T, error)) (T, error) {
	return withDocumentMode(s, path, legacyCompatible, fn)
}

// Every new-version command selects its route under the registry lock. During
// a legacy export, central metadata remains authoritative until registry removal.
// The route is selected under the same registry lock as the metadata load.
// Registered-only callers must refuse before any legacy IO or blocking flock.
func withDocumentMode[T any](s *Sync, path string, mode documentMode, fn func(*file) (T, error)) (T, error) {
	return withRegistry(s, func(t *registryTxn) (T, error) {
		var zero T
		canonical, err := canonicalDocument(path)
		if err != nil {
			return zero, err
		}
		d := t.find(canonical)
		if d == nil {
			if mode == registeredOnly {
				return zero, refuse("%s is not a registered document", path)
			}
			return locked(path, fn)
		}
		raw, err := readRegular(d.File, documentMaxBytes)
		if err != nil {
			return zero, err
		}
		p := t.metadata(*d)
		st, shadow, hasShadow, err := t.loadMetadata(*d)
		if err != nil {
			return zero, err
		}
		lockPaths, err := documentLockPaths(*d, path)
		if err != nil {
			return zero, err
		}
		f := &file{paths: p, content: string(raw), shadow: string(shadow), hasShadow: hasShadow, state: st, owner: EffectiveOwner(string(raw), d.RegisteredOwner), registration: d, lockPaths: lockPaths}
		for _, lockPath := range lockPaths {
			lock, err := readRegular(lockPath, 4096)
			exists := err == nil
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return zero, err
			}
			if ParseHolder(string(lock), exists) == HolderOperator {
				f.holder = HolderOperator
				f.lockText = string(lock)
				break
			}
		}
		return fn(f)
	})
}

// Only spellings still naming this exact canonical document can contribute
// operator locks; a reused old pathname belongs to its new occupant.
func documentLockPaths(d DocumentRegistration, requested string) ([]string, error) {
	abs, err := filepath.Abs(requested)
	if err != nil {
		return nil, err
	}
	candidates := append(documentSpellings(d), abs)
	seen := map[string]bool{}
	var out []string
	for _, path := range candidates {
		if seen[path] {
			continue
		}
		seen[path] = true
		real, err := filepath.EvalSymlinks(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if real == d.File {
			out = append(out, path+".lock")
		}
	}
	return out, nil
}

func (t *registryTxn) exportLegacy(d DocumentRegistration, path string) error {
	_, err := locked(path, func(f *file) (bool, error) {
		st, shadow, exists, err := t.loadMetadata(d)
		if err != nil {
			return false, err
		}
		st.SubmitPaths = cloneRegistryPaths(d.Paths)
		if err := validateRegisteredMetadata(string(shadow), st, d.Paths); err != nil {
			return false, err
		}
		if exists {
			if err := t.publish(f.paths.Shadow, shadow); err != nil {
				return false, err
			}
		} else if err := os.Remove(f.paths.Shadow); err != nil && !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		if raw, err := readRound(t.metadata(d).Round); err == nil && raw != nil {
			if err := t.publish(f.paths.Round, raw); err != nil {
				return false, err
			}
		} else if err := os.Remove(f.paths.Round); err != nil && !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		return true, saveStateUsing(f.paths, st, t.publish)
	})
	return err
}
