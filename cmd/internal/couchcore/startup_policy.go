package couchcore

import (
	"context"
	"errors"
	"sort"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// StartupKind describes the initial fleet evidence, before any startup effects.
type StartupKind string

const (
	StartupFresh        StartupKind = "fresh"
	StartupNormal       StartupKind = "normal"
	StartupFullShutdown StartupKind = "full-shutdown"
	StartupUnknown      StartupKind = "unknown"
)

type StartupCandidate struct {
	Record  ThreadRecord
	Session string
}
type StartupHubObservation struct {
	Candidates []StartupCandidate
	Survived   bool
	Unknown    bool
}

// StartupObservation is ephemeral startup evidence. ParkedCount counts only
// absent sessions with a persisted verified park; ParkedUnknown marks evidence
// that could not be read, rather than calling that evidence absent.
type StartupObservation struct {
	Kind          StartupKind
	Hubs          map[string]StartupHubObservation
	MissingHubs   []string
	ParkedCount   int
	ParkedUnknown bool
	Diagnostic    error `json:"-"`
}

// startupHubRecords uses the same ordinary/managed target projection as the
// switcher. A neighboring path or historical numbered-slot record cannot stand
// in for the selected hub's current conversation.
func startupHubRecords(snapshot ThreadSnapshot, path string) []ThreadRecord {
	// ResolveRepoScope only cleans and hashes the supplied canonical root; no IO.
	scope, err := launcher.ResolveRepoScope(path)
	if err != nil {
		return nil
	}
	records := make(map[ThreadAddress]ThreadRecord, len(snapshot.Records))
	for _, record := range snapshot.Records {
		records[record.Address] = record
	}
	managed := false
	for _, slot := range snapshot.Slots {
		if slot.Identity.WorktreeRoot == path {
			managed = true
			break
		}
	}
	var out []ThreadRecord
	for _, row := range ProjectActionableThreads(FromSnapshot(snapshot, nil)) {
		matches := !managed && primaryOfScope(row, scope.Key)
		if row.Target.Kind == ThreadTargetSlot {
			matches = row.Target.Slot.WorktreeRoot == path
		}
		if matches {
			if record, ok := records[row.Address]; ok {
				out = append(out, record)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastActiveAt.Equal(out[j].LastActiveAt) {
			return out[i].Address.Tag < out[j].Address.Tag
		}
		return out[i].LastActiveAt.After(out[j].LastActiveAt)
	})
	return out
}

// ClassifyStartup is pure: only a positively attributed live session establishes
// normal restart. Unknown and orphaned observations never establish absence.
func ClassifyStartup(snapshot ThreadSnapshot, presence map[ThreadAddress]SessionObservation, hubPaths []string) StartupObservation {
	out := StartupObservation{Kind: StartupFullShutdown, Hubs: map[string]StartupHubObservation{}, ParkedUnknown: len(snapshot.Unreadable) > 0}
	// Even malformed contradictory input must not attribute one terminal to two
	// retained records. The production resolver enforces the same rule.
	claims := map[string]int{}
	for _, record := range snapshot.Records {
		observed := presence[record.Address]
		if observed.State == SessionPresent && observed.Name != "" {
			claims[observed.Name]++
		}
	}
	observedSession := func(address ThreadAddress) SessionObservation {
		observed := presence[address]
		if observed.State == SessionPresent && (observed.Name == "" || claims[observed.Name] != 1) {
			return SessionObservation{State: SessionUnresolved}
		}
		return observed
	}
	survived := false
	for _, slot := range snapshot.Slots {
		if slot.Err != nil {
			out.ParkedUnknown = true
		}
	}
	for _, record := range snapshot.Records {
		observed := observedSession(record.Address)
		switch observed.State {
		case SessionPresent:
			if observed.Name != "" {
				survived = true
			} else {
				out.ParkedUnknown = true
			}
		case SessionAbsent:
			if record.VerifiedPark != nil {
				out.ParkedCount++
			}
		default:
			out.ParkedUnknown = true
		}
	}
	for _, path := range hubPaths {
		if _, exists := out.Hubs[path]; exists {
			continue
		}
		hub := StartupHubObservation{}
		scope, scopeErr := launcher.ResolveRepoScope(path)
		hub.Unknown = scopeErr != nil
		for _, address := range snapshot.Unreadable {
			if address.RepoScope == scope.Key {
				hub.Unknown = true
			}
		}
		for _, slot := range snapshot.Slots {
			if slot.Identity.WorktreeRoot == path && slot.Err != nil {
				hub.Unknown = true
			}
		}
		for _, record := range startupHubRecords(snapshot, path) {
			observed := observedSession(record.Address)
			if observed.State != SessionAbsent && (observed.State != SessionPresent || observed.Name == "") {
				hub.Unknown = true
			}
			if observed.State == SessionPresent && observed.Name != "" {
				hub.Candidates = append(hub.Candidates, StartupCandidate{Record: record, Session: observed.Name})
			}
		}
		if hub.Unknown {
			out.ParkedUnknown = true
		}
		hub.Survived = len(hub.Candidates) > 0
		out.Hubs[path] = hub
		if !hub.Survived {
			out.MissingHubs = append(out.MissingHubs, path)
		}
	}
	if survived {
		out.Kind = StartupNormal
	} else if out.ParkedUnknown {
		out.Kind = StartupUnknown
	} else if len(snapshot.Records) == 0 {
		out.Kind = StartupFresh
	}
	if out.ParkedUnknown {
		out.Diagnostic = errors.New("startup includes unreadable or unresolved session evidence")
	}
	return out
}

// ObserveStartupContext takes one read-only record snapshot and one bulk
// session observation. Diagnostics degrade to unknown; cancellation stays an
// error so a canceled startup cannot continue using incomplete evidence.
func (c *Couch) ObserveStartupContext(ctx context.Context, hubPaths []string) (StartupObservation, error) {
	if err := ctx.Err(); err != nil {
		return StartupObservation{}, err
	}
	snapshot, err := c.Threads.PreviewSnapshotContext(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return StartupObservation{}, ctx.Err()
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return StartupObservation{}, err
		}
		out := ClassifyStartup(ThreadSnapshot{}, nil, hubPaths)
		out.Kind, out.ParkedUnknown, out.Diagnostic = StartupUnknown, true, err
		for path, hub := range out.Hubs {
			hub.Unknown = true
			out.Hubs[path] = hub
		}
		return out, nil
	}
	presence := map[ThreadAddress]SessionObservation{}
	var diagnostic error
	if len(snapshot.Records) > 0 {
		if resolver, ok := c.Artifacts.(SessionPresenceResolver); ok {
			addresses := make([]ThreadAddress, 0, len(snapshot.Records))
			for _, record := range snapshot.Records {
				addresses = append(addresses, record.Address)
			}
			presence, diagnostic = resolver.SessionPresence(ctx, addresses)
			if diagnostic != nil {
				presence = nil
			}
		} else {
			diagnostic = errors.New("startup session observation unavailable")
		}
	}
	if err := ctx.Err(); err != nil {
		return StartupObservation{}, err
	}
	if errors.Is(diagnostic, context.Canceled) || errors.Is(diagnostic, context.DeadlineExceeded) {
		return StartupObservation{}, diagnostic
	}
	out := ClassifyStartup(snapshot, presence, hubPaths)
	if out.Kind == StartupFresh {
		retained, historyErr := c.observeStartupHistory(ctx, hubPaths)
		if historyErr != nil {
			if ctx.Err() != nil {
				return StartupObservation{}, ctx.Err()
			}
			out.Kind, out.ParkedUnknown, out.Diagnostic = StartupUnknown, true, historyErr
		} else if retained {
			out.Kind = StartupFullShutdown
		}
	}
	if diagnostic != nil {
		out.Diagnostic = diagnostic
	}
	return out, nil
}
