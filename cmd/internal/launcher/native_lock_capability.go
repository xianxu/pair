package launcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/readiness"
)

// LegacyLockObservation is current wrapper evidence, never a historical ledger
// row. Live must come from positively observed terminal/registry liveness.
type LegacyLockObservation struct {
	Session, Label, Nonce string
	Live                  bool
	NativeLockVersion     int
}

// LegacyLockSlots returns the distinct live wrappers that predate native locks.
func LegacyLockSlots(observations []LegacyLockObservation) []string {
	labels := map[string]bool{}
	for _, o := range observations {
		if !o.Live || o.NativeLockVersion >= readiness.NativeLockVersion {
			continue
		}
		label := o.Label
		if label == "" {
			label = o.Session
		}
		if label != "" {
			labels[label] = true
		}
	}
	var out []string
	for label := range labels {
		out = append(out, label)
	}
	sort.Strings(out)
	return out
}

// NativeLockExemption is only for the exact source generation of a confirmed
// Relaunch. Ordinary Resume passes the zero value.
type NativeLockExemption struct{ Session, Nonce string }

type nativeLockCapabilityIO struct {
	live    func(context.Context) ([]Session, error)
	read    func(context.Context, string, int64) ([]byte, error)
	readDir func(string) ([]os.DirEntry, error)
}

// CheckNativeLockCapability requires a maintenance cutover before launching a
// native-lock wrapper. It reads host-wide Pair terminal liveness and current
// readiness only, without process inspection or native history reconstruction.
// Matching readiness is the existing current-generation association assumption;
// this check does not introduce independent nonce authority.
func CheckNativeLockCapability(ctx context.Context, globalDataDir string, exemption NativeLockExemption) error {
	return checkNativeLockCapability(ctx, globalDataDir, exemption, nativeLockCapabilityIO{
		live:    (ZellijSource{}).LivenessContext,
		read:    osfs.ReadRegularMetadata,
		readDir: os.ReadDir,
	})
}

func checkNativeLockCapability(ctx context.Context, root string, exemption NativeLockExemption, io nativeLockCapabilityIO) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	legacy, err := artifactpath.ResolveLegacyRoot(root)
	if err != nil {
		return err
	}
	sessions, err := io.live(ctx)
	if err != nil {
		return fmt.Errorf("native lock capability: observe live slots: %w", err)
	}
	live := map[string]bool{}
	for _, s := range sessions {
		if isPairSessionName(s.Name) && s.State != SessionExited {
			live[s.Name] = true
		}
	}
	if len(live) == 0 {
		return nil
	}
	// Resolve the namespace through artifactpath rather than rebuilding sidecars.
	probe, err := artifactpath.ResolveScopeDir(root, "scope")
	if err != nil {
		return err
	}
	scopes, err := io.readDir(filepath.Dir(probe))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("native lock capability: list scopes: %w", err)
	}
	directories := []string{root}
	for _, scope := range scopes {
		if !scope.IsDir() {
			continue
		}
		dir, err := artifactpath.ResolveScopeDir(root, scope.Name())
		if err != nil {
			return err
		}
		directories = append(directories, dir)
	}
	type binding struct {
		entry SessionNameEntry
		dir   string
	}
	bindings := map[string][]binding{}
	for _, dir := range directories {
		if err := ctx.Err(); err != nil {
			return err
		}
		scope, err := artifactpath.ResolveSelectedScope(dir)
		if err != nil {
			return err
		}
		indexPath := scope.SessionBindings()
		if dir == root {
			indexPath = legacy.SessionBindings()
		}
		raw, err := io.read(ctx, indexPath, 4<<20)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("native lock capability: read index: %w", err)
		}
		index, err := DecodeSessionNameIndex(string(raw))
		if err != nil {
			return fmt.Errorf("native lock capability: %w", err)
		}
		for name := range live {
			if entry, ok := index.ownerOf(name); ok {
				bindings[name] = append(bindings[name], binding{entry, dir})
			}
		}
	}
	var observations []LegacyLockObservation
	for name := range live {
		matched := false
		label := name
		for _, binding := range bindings[name] {
			entry := binding.entry
			label = entry.RepoName + "/" + entry.Tag
			paths, err := artifactpath.ResolveScoped(binding.dir, entry.Tag)
			if err != nil {
				return err
			}
			for _, agent := range AgentInventory() {
				if err := ctx.Err(); err != nil {
					return err
				}
				raw, err := io.read(ctx, paths.AgentReady(agent), 64<<10)
				if os.IsNotExist(err) {
					continue
				}
				if err != nil {
					return fmt.Errorf("native lock capability: read readiness for %s: %w", label, err)
				}
				record, err := readiness.Decode(string(raw))
				if err != nil {
					return fmt.Errorf("native lock capability: decode readiness for %s: %w", label, err)
				}
				if record.Session != name || record.Tag != entry.Tag || record.Agent != agent {
					continue
				}
				matched = true
				if exemption.Session == name && exemption.Nonce != "" && exemption.Nonce == record.Nonce {
					continue
				}
				observations = append(observations, LegacyLockObservation{Session: name, Label: label, Nonce: record.Nonce, Live: true, NativeLockVersion: record.NativeLockVersion})
			}
		}
		if !matched {
			observations = append(observations, LegacyLockObservation{Session: name, Label: label, Live: true})
		}
	}
	if blockers := LegacyLockSlots(observations); len(blockers) > 0 {
		return fmt.Errorf("relaunch older slots first: %s", strings.Join(blockers, ", "))
	}
	return nil
}
