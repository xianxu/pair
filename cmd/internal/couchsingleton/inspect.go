package couchsingleton

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func sameStrings(a, b []string) bool {
	a = append([]string(nil), a...)
	b = append([]string(nil), b...)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}
func readBounded(path string, limit int64) ([]byte, error) {
	fd, e := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return nil, e
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", path)
	}
	b, e := io.ReadAll(io.LimitReader(f, limit+1))
	if e == nil && int64(len(b)) > limit {
		e = fmt.Errorf("file exceeds %d bytes: %s", limit, path)
	}
	return b, e
}
func resolved(defaults Roots, q Request) Roots {
	r := defaults
	if q.Roots.StoreDir != "" {
		r.StoreDir = q.Roots.StoreDir
	}
	if q.Roots.PairDataDir != "" {
		r.PairDataDir = q.Roots.PairDataDir
	}
	if q.Roots.IdentityDir != "" {
		r.IdentityDir = q.Roots.IdentityDir
	}
	return r
}

// canonicalMissing resolves existing ancestors without creating any directory.
func canonicalMissing(p string) error {
	if !filepath.IsAbs(p) || filepath.Clean(p) != p {
		return fmt.Errorf("noncanonical path %q", p)
	}
	for at := p; ; at = filepath.Dir(at) {
		_, e := os.Lstat(at)
		if errors.Is(e, os.ErrNotExist) {
			continue
		}
		if e != nil {
			return e
		}
		physical, e := filepath.EvalSymlinks(at)
		if e != nil {
			return e
		}
		if physical != at {
			return fmt.Errorf("path changed physical identity: %s", at)
		}
		st, e := os.Stat(at)
		if e != nil {
			return e
		}
		if !st.IsDir() {
			return fmt.Errorf("not a directory: %s", at)
		}
		return nil
	}
}
func (m Manager) inspect(q Request, owned *couchcore.SupervisorLease) (Report, error) {
	return m.inspectSources(q, owned, nil)
}
func (m Manager) inspectSources(q Request, owned *couchcore.SupervisorLease, inspection *couchcore.StoreInspection) (Report, error) {
	report := Report{Status: "READY"}
	r := resolved(m.Defaults, q)
	if e := validateRoots(r); e != nil {
		return report, e
	}
	for _, p := range []string{r.StoreDir, r.PairDataDir, r.IdentityDir} {
		if e := canonicalMissing(p); e != nil {
			return report, e
		}
	}
	paths := map[string]bool{r.StoreDir: false, m.Defaults.StoreDir: false}
	associated := map[string]bool{filepath.Join(r.PairDataDir, "couch"): true}
	if q.Roots.StoreDir != "" && q.Roots.PairDataDir != "" {
		associated[q.Roots.StoreDir] = true
	}
	for _, p := range q.Stores {
		paths[p] = true
	}
	for _, p := range q.Exclude {
		paths[p] = true
	}
	identity, e := (couchidentity.IdentityStore{HostDir: r.IdentityDir, StoreDir: r.StoreDir}).Inspect()
	if e != nil {
		report.Blockers = append(report.Blockers, e.Error())
	}
	for _, entry := range identity.Registrations {
		paths[entry.StorePath] = true
	}
	evidence := map[string]string{}
	registryPath := filepath.Join(r.PairDataDir, ".retention", "stores.json")
	if b, e := readBounded(registryPath, 4<<20); e == nil {
		evidence[registryPath] = string(b)
		var registry storagegc.StoreRegistry
		if err := strictjson.Decode(b, &registry); err != nil {
			return report, err
		}
		if len(registry.Stores) > 4096 {
			return report, errors.New("too many registered Couch stores")
		}
		// Validate and enumerate the same captured bytes that enter the digest.
		// A filesystem reread here could probe a different registry version.
		if err := registry.ValidateStructure(); err != nil {
			return report, err
		}
		if err := m.validatePaths(r, registry.Stores); err != nil {
			return report, err
		}
		for _, path := range registry.Stores {
			paths[path] = true
			associated[path] = true
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		report.Blockers = append(report.Blockers, e.Error())
	}
	hp := filepath.Join(r.IdentityDir, "couch-identities.json")
	if b, e := readBounded(hp, 4<<20); e == nil {
		evidence[hp] = string(b)
	} else if !errors.Is(e, os.ErrNotExist) {
		report.Blockers = append(report.Blockers, e.Error())
	}
	if len(paths) > 4096 {
		return report, errors.New("too many registered Couch stores")
	}
	ordered := make([]string, 0, len(paths))
	for p := range paths {
		ordered = append(ordered, p)
	}
	slices.Sort(ordered)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	budget := &inspectionBudget{ctx: ctx}
	processes := map[string][]couchcore.RecordedProcessObservation{}
	for _, p := range ordered {
		if e := ctx.Err(); e != nil {
			return report, e
		}
		if e := m.validatePaths(r, []string{p}); e != nil {
			return report, e
		}
		c := Candidate{Roots: r, State: "empty", Registered: paths[p]}
		c.Roots.StoreDir = p
		if e := canonicalMissing(p); e != nil {
			c.State = "unknown"
			c.Problem = e.Error()
		} else if _, e := os.Lstat(p); errors.Is(e, os.ErrNotExist) {
			c.State = "missing"
		} else if e != nil {
			c.State = "unknown"
			c.Problem = e.Error()
		} else {
			ns, e := couchcore.ExistingCouchNamespace(p)
			if e != nil {
				c.Problem = e.Error()
			} else {
				observation, e := couchcore.ObserveSupervisor(ns, m.Proc, owned)
				if e != nil {
					c.Owner = "unknown: " + e.Error()
				} else if observation.Held {
					c.Owner = fmt.Sprintf("live supervisor %+v", observation.Owner)
				}

				id, e := (couchidentity.IdentityStore{HostDir: r.IdentityDir, StoreDir: p}).Inspect()
				if e != nil {
					c.Problem = e.Error()
				}
				if id.LocalExists {
					c.State = "populated"
				}

				observe := func(held *couchcore.StoreInspection) error {
					n, e := hashTree(p, evidence, budget)
					if e != nil {
						return e
					}
					if n > 0 {
						c.State = "populated"
					}
					snapshot, e := held.Snapshot(ns)
					if e != nil {
						return e
					}
					if len(snapshot.Unreadable) > 0 {
						return fmt.Errorf("unreadable thread records in %s", p)
					}
					for _, slot := range snapshot.Slots {
						root := filepath.Join(slot.Identity.EnvironmentRoot, ".couch")
						if slot.Err != nil {
							return fmt.Errorf("unresolved slot %s: %w", root, slot.Err)
						}
						if e := m.validatePaths(r, []string{root}); e != nil {
							return e
						}
						raw, e := json.Marshal(slot.Identity)
						if e != nil {
							return e
						}
						evidence["slot:"+root] = string(raw)
						if _, e := os.Lstat(root); errors.Is(e, os.ErrNotExist) {
							evidence["slot-state:"+root] = "missing"
							continue
						} else if e != nil {
							return e
						}
						evidence["slot-state:"+root] = "present"
						if _, e := hashTree(root, evidence, budget); e != nil {
							return e
						}
					}
					var processErr error
					processes[p], processErr = couchcore.ObserveMigrationProcesses(ctx, m.Proc, snapshot.Records)
					return processErr
				}
				var inspectErr error
				if inspection != nil {
					inspectErr = observe(inspection)
				} else {
					inspectErr = couchcore.WithStoreInspectionLocks(ctx, []couchcore.CouchNamespace{ns}, func(path string) error { return m.validatePaths(r, []string{path}) }, observe)
				}
				if inspectErr != nil {
					c.Problem = fmt.Sprintf("inspect store %s: %v", p, inspectErr)
				}
			}
		}
		report.Candidates = append(report.Candidates, c)
	}
	selection, e := DecideAdoption(q, r, report.Candidates)
	report.Selection = selection
	if e == nil {
		for _, path := range ordered {
			observations := processes[path]
			if path == selection.Roots.StoreDir {
				continue
			}
			for _, observation := range observations {
				if observation.Liveness != couchcore.Dead {
					report.Blockers = append(report.Blockers, fmt.Sprintf("UNMIGRATED: store %q has %s incarnation pid %d identity %q; stop it or establish absence before exclusion", path, observation.Liveness, observation.Process.PID, observation.Process.Identity))
				}
			}
		}
	}

	if e == nil && !associated[selection.Roots.StoreDir] {
		for _, c := range report.Candidates {
			if c.Roots.StoreDir == selection.Roots.StoreDir && c.State == "populated" {
				e = fmt.Errorf("UNMIGRATED: supporting Pair root for %q is unknown; supply --adopt-store with --pair-data and the existing --identity-dir", selection.Roots.StoreDir)
			}
		}
	}
	if e != nil {
		report.Blockers = append(report.Blockers, e.Error())
	}
	if len(report.Blockers) > 0 {
		report.Status = "UNMIGRATED"
	}
	// Ownership is a separate admission predicate. Missing empty roots and roots
	// created after admission have identical preservation evidence.
	raw, _ := json.Marshal(struct {
		Request  Request
		Roots    Roots
		Evidence map[string]string
	}{q, r, evidence})
	sum := sha256.Sum256(raw)
	report.Digest = hex.EncodeToString(sum[:])
	return report, nil
}

type inspectionBudget struct {
	ctx            context.Context
	entries, bytes int
}

func hashTree(root string, evidence map[string]string, budget *inspectionBudget) (int, error) {
	count := 0
	e := filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if e := budget.ctx.Err(); e != nil {
			return e
		}
		budget.entries++
		if budget.entries > 65536 {
			return errors.New("adoption inspection exceeds 65536 directory entries")
		}
		if p == root {
			return nil
		}
		name := d.Name()
		if name == "supervisor.lock" || name == "supervisor-owner.json" || strings.HasPrefix(name, ".supervisor-owner-") || name == "store.lock" || name == "identities.lock" {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		count++

		b, e := readBounded(p, 4<<20)
		if e != nil {
			return e
		}
		budget.bytes += len(b)
		if budget.bytes > 64<<20 {
			return errors.New("store inspection byte limit exceeded")
		}
		sum := sha256.Sum256(b)
		evidence[p] = hex.EncodeToString(sum[:])
		return nil
	})
	return count, e
}

func (m Manager) validateRequest(q Request) error {
	if len(q.Stores) > 4096 || len(q.Exclude) > 4096 {
		return errors.New("explicit inventory exceeds 4096 stores")
	}
	if e := validateRoots(resolved(m.Defaults, q)); e != nil {
		return e
	}
	if e := canonicalMissing(m.AuthorityDir); e != nil {
		return e
	}
	return m.validatePaths(resolved(m.Defaults, q), append(append([]string{m.AuthorityDir}, q.Stores...), q.Exclude...))
}
func (m Manager) validatePaths(r Roots, extra []string) error {
	if m.IsolationRoot == "" {
		return nil
	}
	if e := canonicalMissing(m.IsolationRoot); e != nil {
		return e
	}
	for _, p := range append([]string{r.StoreDir, r.PairDataDir, r.IdentityDir}, extra...) {
		rel, e := filepath.Rel(m.IsolationRoot, p)
		if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("path %q escapes isolated root %q", p, m.IsolationRoot)
		}
		if e := canonicalMissing(p); e != nil {
			return e
		}
	}
	return nil
}
