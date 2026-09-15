package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// RetentionOps is optional for portable runtimes; the production OSRuntime
// always implements it. It reserves storage before selected artifact access.
type RetentionOps interface {
	BeginRetention(dataDir, tag string, create bool) (RetentionUse, error)
}
type RetentionUse interface {
	ReservationID() string
	BeforeSpawn() error
	Finish(success bool) error
}

func beginRetention(rt Runtime, dataDir, tag string, create bool) (RetentionUse, error) {
	ops, ok := rt.(RetentionOps)
	if !ok {
		return nil, nil
	}
	use, err := ops.BeginRetention(dataDir, tag, create)
	if err != nil {
		return nil, err
	}
	if use == nil {
		return nil, errors.New("retention runtime returned no guard")
	}
	rt.SetEnv("PAIR_RETENTION_PROTOCOL", "1")
	rt.SetEnv("PAIR_RETENTION_START_ID", use.ReservationID())
	return use, nil
}

type launchRetention struct {
	coordinator         *storagegc.Coordinator
	owner               artifactpath.StorageOwner
	parent              storagegc.ProcessIdentity
	registration, start string
	spawned             bool
	background          bool
}

func (u *launchRetention) ReservationID() string { return u.start }
func (u *launchRetention) BeforeSpawn() error {
	if u.start == "" || u.spawned {
		return nil
	}
	if err := u.coordinator.MarkStartSpawned(context.Background(), u.owner, u.start, u.parent); err != nil {
		return err
	}
	u.spawned = true
	return nil
}
func (u *launchRetention) Finish(success bool) (err error) {
	ctx := context.Background()
	if success && !u.background {
		id, e := u.coordinator.BeginUse(ctx, u.owner, u.parent, "selected-launch")
		if e == nil {
			e = u.coordinator.CompleteUse(ctx, u.owner, id)
		}
		err = errors.Join(err, e)
	}
	if u.start != "" && !u.spawned {
		err = errors.Join(err, u.coordinator.CancelStartBeforeSpawn(ctx, u.owner, u.start, u.parent))
	}
	// Failure after spawning deliberately keeps the reservation. Parent death or
	// a failed child wait cannot prove detached children made no effects.
	err = errors.Join(err, u.coordinator.ReleaseProcess(ctx, u.owner, u.registration))
	return err
}

func (r OSRuntime) BeginRetention(dataDir, tag string, create bool) (RetentionUse, error) {
	if !filepath.IsAbs(r.GlobalDataDir) || !filepath.IsAbs(dataDir) {
		return nil, errors.New("retention needs explicit absolute data roots")
	}
	retentionRoot := r.GlobalDataDir
	relative, err := filepath.Rel(retentionRoot, dataDir)
	if err != nil {
		return nil, err
	}
	scope := ""
	if relative != "." {
		parts := strings.Split(relative, string(filepath.Separator))
		if len(parts) != 2 || parts[0] != "repos" {
			if filepath.Clean(dataDir) != filepath.Clean(r.DataDir) {
				return nil, errors.New("selected data directory is outside Pair namespace")
			}
			retentionRoot = dataDir
			if filepath.Base(filepath.Dir(dataDir)) == "repos" {
				scope = filepath.Base(dataDir)
				retentionRoot = filepath.Dir(filepath.Dir(dataDir))
			}
		} else {
			scope = parts[1]
		}
	}
	if _, err := artifactpath.NewStorageOwner(retentionRoot, scope, tag); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(retentionRoot, 0700); err != nil {
		return nil, err
	}
	c, err := storagegc.NewCoordinator(retentionRoot)
	if err != nil {
		return nil, err
	}
	owner, err := artifactpath.NewStorageOwner(c.Root, scope, tag)
	if err != nil {
		return nil, err
	}
	if scope != "" {
		for _, dir := range []string{filepath.Join(c.Root, "repos"), owner.Directory()} {
			if err := os.Mkdir(dir, 0700); err != nil && !errors.Is(err, os.ErrExist) {
				return nil, err
			}
			st, err := os.Lstat(dir)
			if err != nil {
				return nil, err
			}
			if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
				return nil, errors.New("unsafe retention namespace directory")
			}
		}
	}
	selected, err := filepath.EvalSymlinks(dataDir)
	if err != nil || selected != owner.Directory() {
		return nil, errors.New("selected physical directory escapes Pair owner")
	}
	parent, err := storagegc.CurrentProcessIdentity(os.Getpid())
	if err != nil {
		return nil, err
	}
	registration, err := c.RegisterProcess(context.Background(), owner, parent, "launcher")
	if err != nil {
		return nil, err
	}
	use := &launchRetention{coordinator: c, owner: owner, parent: parent, registration: registration, background: os.Getenv("PAIR_RETENTION_BACKGROUND") == "1"}
	r.SetEnv("PAIR_RETENTION_BACKGROUND", "")
	if create {
		use.start, err = c.ReserveStart(context.Background(), owner, parent, []string{"wrapper", "draft-editor"})
		if err != nil {
			return nil, errors.Join(err, c.ReleaseProcess(context.Background(), owner, registration))
		}
	}
	r.SetEnv("PAIR_SCOPE_KEY", owner.RepoScope)
	return use, nil
}

func finishLaunchRetention(step launchStep, stderr io.Writer) bool {
	if step.retention == nil {
		return true
	}
	if err := step.retention.Finish(step.code == 0); err != nil {
		fmt.Fprintf(stderr, "pair: retention completion failed for '%s': %v\n", step.tag, err)
		return false
	}
	return true
}

func (u *launchRetention) WriteChanged(target string, write func() (bool, error)) error {
	return u.coordinator.WriteChanged(context.Background(), u.owner, u.parent, target, write)
}
func writeRetainedDraft(rt Runtime, use RetentionUse, path, text string) error {
	prior, err := rt.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if prior == text {
		return nil
	}
	write := func() (bool, error) {
		if err := rt.WriteAtomic(path, text); err != nil {
			return false, err
		}
		return true, nil
	}
	if writer, ok := use.(interface {
		WriteChanged(string, func() (bool, error)) error
	}); ok {
		return writer.WriteChanged(path, write)
	}
	_, err = write()
	return err
}

func retainRestartOwners(rt Runtime, dataDir string, step launchStep, marker RestartMarker) ([]RetentionUse, error) {
	seen := map[string]bool{step.tag: true}
	var uses []RetentionUse
	for _, tag := range []string{marker.Tag, marker.RenameTo} {
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		use, err := beginRetention(rt, dataDir, tag, false)
		if err != nil {
			for _, held := range uses {
				err = errors.Join(err, held.Finish(false))
			}
			return nil, err
		}
		if use != nil {
			uses = append(uses, use)
		}
	}
	return uses, nil
}
func finishRestartOwners(uses []RetentionUse, stderr io.Writer) bool {
	ok := true
	for _, use := range uses {
		if err := use.Finish(false); err != nil {
			fmt.Fprintf(stderr, "pair: restart retention release: %v\n", err)
			ok = false
		}
	}
	return ok
}
