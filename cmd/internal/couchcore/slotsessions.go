package couchcore

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

// SlotSessionCandidateSource enumerates the entire managed scope, including
// addresses no longer present in current Couch metadata.
type SlotSessionCandidateSource interface {
	SlotSessionCandidates(context.Context, string) ([]ThreadAddress, error)
}
type SlotSessionCandidate struct {
	Address   ThreadAddress
	Record    *ThreadRecord
	Presence  SessionState
	Processes []ProcessIdentity
}
type SlotSessionObservation struct {
	Candidates []SlotSessionCandidate
	Absent     bool
}

func (c ScopedThreadArtifactCollisionChecker) SlotSessionCandidates(ctx context.Context, scope string) ([]ThreadAddress, error) {
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: c.GlobalDataDir, RepoScope: scope, Tag: "validation"})
	if err != nil {
		return nil, err
	}
	if err := provisionSafePath(paths.ScopeDir()); err != nil {
		return nil, err
	}
	for _, dir := range []string{c.GlobalDataDir, paths.ScopeDir()} {
		selected, err := artifactpath.ResolveSelectedScope(dir)
		if err != nil {
			return nil, err
		}
		if err := provisionSafePath(selected.SessionBindings()); err != nil {
			return nil, err
		}
	}
	index, err := launcher.NewScopedOSRuntime(c.GlobalDataDir, paths.ScopeDir(), "").ReadSessionNameIndex()
	if err != nil {
		return nil, err
	}
	addresses := map[ThreadAddress]bool{}
	for _, entry := range index.Entries {
		if entry.ScopeKey == scope {
			addresses[ThreadAddress{RepoScope: scope, Tag: ThreadTag(entry.Tag)}] = true
		}
	}
	entries, err := os.ReadDir(paths.ScopeDir())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if len(entries) > 16384 {
		return nil, errors.New("slot native scope exceeds 16384 entries; export obsolete artifacts before recovery")
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("symlink in slot native scope: %s", entry.Name())
		}
		names = append(names, entry.Name())
		// Pair's no-replace address marker anchors owners before any session or
		// history artifact exists. Validate the recognized filename by its owner.
		if strings.HasPrefix(entry.Name(), "thread-claim-") && strings.HasSuffix(entry.Name(), ".json") {
			tag := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "thread-claim-"), ".json")
			address := ThreadAddress{RepoScope: scope, Tag: ThreadTag(tag)}
			if err := validateThreadAddress(address); err != nil {
				return nil, err
			}
			exact, err := artifactpath.Resolve(artifactpath.Address{DataDir: c.GlobalDataDir, RepoScope: scope, Tag: tag})
			if err != nil {
				return nil, err
			}
			if exact.ThreadClaim() != filepath.Join(paths.ScopeDir(), entry.Name()) {
				return nil, errors.New("noncanonical native owner")
			}
			addresses[address] = true
		}
	}
	owners, err := artifactpath.DiscoverStorageOwners(c.GlobalDataDir, scope, names)
	if err != nil {
		return nil, err
	}
	for _, owner := range owners {
		addresses[ThreadAddress{RepoScope: scope, Tag: ThreadTag(owner.Tag)}] = true
	}
	return sortedSlotAddresses(addresses), ctx.Err()
}
func sortedSlotAddresses(set map[ThreadAddress]bool) []ThreadAddress {
	out := make([]ThreadAddress, 0, len(set))
	for address := range set {
		out = append(out, address)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RepoScope != out[j].RepoScope {
			return out[i].RepoScope < out[j].RepoScope
		}
		return out[i].Tag < out[j].Tag
	})
	return out
}
func (f *FakeThreadArtifactCollisionChecker) SlotSessionCandidates(ctx context.Context, scope string) ([]ThreadAddress, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	set := map[ThreadAddress]bool{}
	add := func(a ThreadAddress) {
		if a.RepoScope == scope {
			set[a] = true
		}
	}
	for a := range f.values {
		add(a)
	}
	for a := range f.registrations {
		add(a)
	}
	for a := range f.pairSessions {
		add(a)
	}
	for a := range f.sessionPresence {
		add(a)
	}
	for a := range f.detachedSessions {
		add(a)
	}
	for k := range f.nativeBindings {
		add(k.Address)
	}
	return sortedSlotAddresses(set), nil
}

func (c *Couch) ObserveSlotSessions(ctx context.Context, slot SlotIdentity) (SlotSessionObservation, error) {
	var out SlotSessionObservation
	if err := slot.Validate(); err != nil {
		return out, err
	}
	if c.Proc == nil || c.Threads == nil {
		return out, errors.New("slot ownership observers unavailable")
	}
	source, ok := c.Artifacts.(SlotSessionCandidateSource)
	if !ok {
		return out, errors.New("complete slot native scope scanner unavailable")
	}
	presence, ok := c.Artifacts.(SessionPresenceResolver)
	if !ok {
		return out, errors.New("slot session presence observer unavailable")
	}
	scope, err := launcher.ResolveRepoScope(slot.WorktreeRoot)
	if err != nil {
		return out, err
	}
	addresses, err := source.SlotSessionCandidates(ctx, scope.Key)
	if err != nil {
		return out, err
	}
	candidates := map[ThreadAddress]*SlotSessionCandidate{}
	add := func(address ThreadAddress) (*SlotSessionCandidate, error) {
		if address.RepoScope != scope.Key {
			return nil, errors.New("slot native owner belongs to another scope")
		}
		if err := validateThreadAddress(address); err != nil {
			return nil, err
		}
		if candidates[address] == nil {
			candidates[address] = &SlotSessionCandidate{Address: address}
		}
		return candidates[address], nil
	}
	for _, address := range addresses {
		if _, err := add(address); err != nil {
			return out, err
		}
	}
	local := newSlotThreadStore(c.Namespace, slot)
	local.coordinator = c.Threads.coordinator
	err = local.withLock(func() error {
		current, err := local.readSlotCurrentLocked()
		if err != nil {
			return err
		}
		if current.Unsupported {
			return current.Err
		}
		addRecord := func(record ThreadRecord) error {
			candidate, err := add(record.Address)
			if err != nil {
				return err
			}
			candidate.Record = &record
			for _, inc := range record.Incarnations {
				if inc.PID > 0 {
					candidate.Processes = append(candidate.Processes, ProcessIdentity{PID: inc.PID, Identity: inc.Identity})
				}
				if inc.Start != nil {
					candidate.Processes = append(candidate.Processes, ProcessIdentity{PID: inc.Start.OwnerPID, Identity: inc.Start.OwnerIdentity})
				}
			}
			return nil
		}
		if current.Record != nil {
			if err := addRecord(*current.Record); err != nil {
				return err
			}
		}
		root := filepath.Join(local.root, "archive")
		count := 0
		archiveBytes := 0
		return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if errors.Is(walkErr, os.ErrNotExist) && path == root {
				return nil
			}
			if walkErr != nil {
				return walkErr
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return errors.New("symlink in slot archive ownership")
			}
			if entry.IsDir() {
				return nil
			}
			count++
			if count > 4096 {
				return errors.New("slot archive ownership exceeds 4096 records")
			}
			relative, _ := filepath.Rel(root, path)
			parts := strings.Split(relative, string(filepath.Separator))
			if len(parts) != 2 || !strings.HasSuffix(parts[1], ".json") {
				return errors.New("unknown slot archive owner")
			}
			address := ThreadAddress{RepoScope: parts[0], Tag: ThreadTag(strings.TrimSuffix(parts[1], ".json"))}
			if _, err := add(address); err != nil {
				return err
			}
			raw, err := local.readRetentionFile(path)
			if err != nil {
				return err
			}
			archiveBytes += len(raw)
			if archiveBytes > 64<<20 {
				return errors.New("slot archive observation exceeds 64 MiB; export old history before recovery")
			}
			record, err := local.decodeThreadRaw(address, raw)
			if err == nil {
				return addRecord(record)
			}
			return nil
		})
	})
	if err != nil {
		return out, err
	}
	for _, actor := range c.reg.Records() {
		if actor.Args.WorkingDir() != slot.WorktreeRoot && actor.Thread.RepoScope != scope.Key {
			continue
		}
		candidate, err := add(actor.Thread)
		if err != nil {
			return out, err
		}
		candidate.Processes = append(candidate.Processes, ProcessIdentity{PID: actor.PID, Identity: actor.Identity})
		if candidate.Record == nil && actor.Args.WorkingDir() == slot.WorktreeRoot && launcher.IsSupportedAgent(actor.Args.Stack) {
			record := ThreadRecord{SchemaVersion: ThreadSchemaVersion, Address: actor.Thread, StartingPath: slot.WorktreeRoot, WorkingPath: slot.WorktreeRoot, CreatedAt: actor.StartedAt, Revision: 1, LatestLaunchProfile: &LaunchProfile{Agent: actor.Args.Stack, Argv: cloneArgv(actor.Args.ExtraArgs)}}
			candidate.Record = &record
		}
	}
	set := map[ThreadAddress]bool{}
	for address := range candidates {
		set[address] = true
	}
	addresses = sortedSlotAddresses(set)
	if len(addresses) > 4096 {
		return out, errors.New("slot native ownership exceeds 4096 candidates")
	}
	observed, err := presence.SessionPresence(ctx, addresses)
	if err != nil {
		return out, err
	}
	out.Absent = true
	for _, address := range addresses {
		candidate := candidates[address]
		candidate.Presence = observed[address].State
		if candidate.Presence == SessionUnresolved {
			return out, fmt.Errorf("session ownership unresolved for %s", address.Tag)
		}
		if candidate.Presence != SessionAbsent {
			out.Absent = false
		}
		for _, process := range candidate.Processes {
			state := observeExactProcess(c.Proc, process)
			if state == Unknown {
				return out, fmt.Errorf("process ownership unresolved for %s", address.Tag)
			}
			if state == Live {
				out.Absent = false
			}
		}
		if candidate.Record == nil && candidate.Presence == SessionAbsent {
			registered, err := c.Artifacts.Registration(address)
			if err != nil {
				return out, err
			}
			if registered != RegistrationEstablished {
				return out, fmt.Errorf("unowned native claim %s has no stopped-process proof; inspect and remove only its stale claim after confirming its owner stopped", address.Tag)
			}
		}
		out.Candidates = append(out.Candidates, *candidate)
	}
	return out, ctx.Err()
}
