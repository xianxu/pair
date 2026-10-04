package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// FleetInventoryTimeout bounds one `sdlc fleet inventory` run. sdlc's own
// claim reads take up to 15 s each with at most 8 in flight (pair#367).
const FleetInventoryTimeout = 90 * time.Second

// FleetInventorySource yields one fleet's raw `sdlc fleet inventory --json`
// document, read from a vantage checkout inside that fleet. The bytes are
// untrusted; DecodeFleetInventory is their only reader.
type FleetInventorySource interface {
	FleetInventory(ctx context.Context, vantage string) ([]byte, error)
}

// SDLCFleetSource runs sdlc through the shared ProvisionIO seam: its own
// process group, a timeout and the 1 MiB stdout cap.
type SDLCFleetSource struct {
	IO      ProvisionIO
	Timeout time.Duration
}

func (s SDLCFleetSource) FleetInventory(ctx context.Context, vantage string) ([]byte, error) {
	if s.IO == nil {
		return nil, errors.New("sdlc fleet source is unavailable")
	}
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = FleetInventoryTimeout
	}
	return s.IO.Run(ctx, ProvisionCommand{
		Dir: vantage, Program: "sdlc",
		Args:    []string{"fleet", "inventory", "--json", "--path", vantage},
		Timeout: timeout,
	})
}

// SlotGitProbeTimeout bounds one checkout's `git status` read. The console's
// quick-status poll and the recover-plan fallback share it.
const SlotGitProbeTimeout = 3 * time.Second

// maxRecoverFleets caps the sdlc runs one report makes; normally there is one.
const maxRecoverFleets = 8

// RecoverPlan gathers both observations and derives the report (pair#367).
// It runs `sdlc fleet inventory` once per fleet of the enrolled repositories,
// reads the actionable inventory the way `couch --list` gathers evidence, and
// probes git itself only for the slots of a fleet sdlc could not describe.
// Failed sources degrade the rows they touch; only an unreadable enrollment
// list is returned as an error, because without it there is no fleet to ask.
// It writes nothing.
func (c *Couch) RecoverPlan(ctx context.Context) (RecoverPlan, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	names, err := c.repositoryNames()
	if err != nil {
		return RecoverPlan{}, err
	}
	type fleetGroup struct {
		root      string
		primaries []string
		err       string
	}
	groups := map[string]*fleetGroup{}
	var failed []*fleetGroup
	for _, name := range names {
		id, err := c.slotWorkspace(ctx, name.Key)
		if err != nil || id.FleetRoot == "" {
			reason := "workspace probe returned no fleet root"
			if err != nil {
				reason = err.Error()
			}
			failed = append(failed, &fleetGroup{root: name.Key, primaries: []string{name.Key}, err: "resolve fleet: " + reason})
			continue
		}
		g := groups[id.FleetRoot]
		if g == nil {
			g = &fleetGroup{root: id.FleetRoot}
			groups[id.FleetRoot] = g
		}
		g.primaries = append(g.primaries, name.Key)
	}
	roots := make([]string, 0, len(groups))
	for root := range groups {
		roots = append(roots, root)
	}
	sort.Strings(roots)
	var input RecoverPlanInput
	input.LocalGit = map[string]RecoverLocalGit{}
	ordered := make([]*fleetGroup, 0, len(roots)+len(failed))
	for _, root := range roots {
		sort.Strings(groups[root].primaries)
		ordered = append(ordered, groups[root])
	}
	ordered = append(ordered, failed...)
	asked := 0
	for _, g := range ordered {
		obs := FleetObservation{Root: g.root, Vantage: g.primaries[0], State: FleetObservationUnavailable, Error: g.err}
		switch {
		case g.err != "":
		case asked >= maxRecoverFleets:
			obs.Error = "fleet limit"
		case c.Fleet == nil:
			obs.Error = "sdlc fleet source is unavailable"
		default:
			asked++
			raw, err := c.Fleet.FleetInventory(ctx, obs.Vantage)
			if err == nil {
				obs.Inventory, err = DecodeFleetInventory(raw)
			}
			switch {
			case err == nil:
				obs.State = FleetObservationPresent
			case errors.Is(err, ErrFleetSchemaUnsupported):
				obs.State, obs.Error = FleetObservationUnsupported, err.Error()
			default:
				obs.Error = err.Error()
			}
		}
		input.Fleets = append(input.Fleets, obs)
		if obs.State != FleetObservationPresent {
			for _, primary := range g.primaries {
				input.SlotCandidates = append(input.SlotCandidates, c.recoverCandidates(ctx, g.root, primary, input.LocalGit)...)
			}
		}
	}
	rows, err := c.ActionableThreadInventoryContext(ctx, nil)
	if err != nil {
		input.Couch = CouchObservation{State: CouchObservationUnavailable, Error: err.Error()}
	} else {
		input.Couch = CouchObservation{State: CouchObservationOK, Rows: rows}
	}
	return DeriveRecoverPlan(input), nil
}

// recoverCandidates lists one primary's slots by Couch's conventional layout
// and probes each present host's git through ProbeSlotGit, recording a failed
// probe as unknown.
func (c *Couch) recoverCandidates(ctx context.Context, fleet, primary string, probes map[string]RecoverLocalGit) []RecoverSlotCandidate {
	repo := filepath.Base(primary)
	candidates := []RecoverSlotCandidate{{Fleet: fleet, Address: WorkspaceReference{Repo: repo}.String(), Path: primary}}
	if _, err := os.Stat(primary); err != nil {
		candidates[0].Missing = true
	} else if slots, err := EnumerateSlotCandidates(primary); err == nil {
		for _, slot := range slots {
			candidates = append(candidates, RecoverSlotCandidate{Fleet: fleet, Address: WorkspaceReference{Repo: repo, Number: slot.Identity.Number}.String(),
				Path: slot.Identity.WorktreeRoot, Missing: slot.Err != nil})
		}
	}
	for _, candidate := range candidates {
		if candidate.Missing {
			continue
		}
		if c.Git == nil {
			probes[candidate.Path] = RecoverLocalGit{Err: "git is unavailable"}
			continue
		}
		probeCtx, cancel := context.WithTimeout(ctx, SlotGitProbeTimeout)
		status, err := ProbeSlotGit(probeCtx, c.Git, candidate.Path)
		cancel()
		if err != nil {
			probes[candidate.Path] = RecoverLocalGit{Err: err.Error()}
			continue
		}
		probes[candidate.Path] = RecoverLocalGit{Status: status}
	}
	return candidates
}
