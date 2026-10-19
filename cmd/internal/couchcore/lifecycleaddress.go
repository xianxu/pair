package couchcore

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// AddressKind names the two things a lifecycle operation can act on
// (pair#695 §3): a slot's checkout, or a team hub's folder.
type AddressKind string

const (
	AddressSlot AddressKind = "slot"
	AddressTeam AddressKind = "team"
)

// TeamIdentity is a team hub: a folder holding a committed .hub inside an
// operations checkout. Several teams may share one checkout. Scope and Name
// are resolved by NewTeamAddress, so Key stays pure.
type TeamIdentity struct {
	CheckoutRoot string // physical worktree root of the operations checkout
	TeamDir      string // clean relative path to the team folder, e.g. teams/devinfra
	Name         string // filepath.Base(TeamDir)
	Scope        string // launcher.ResolveRepoScope(CheckoutRoot).Key
}

// LifecycleAddress is what the lifecycle primitives act on: the lease, the
// thread store, failure records and orphan origins are all keyed by Key().
// Exactly one arm is set, matching Kind. Callers build it with NewSlotAddress
// or NewTeamAddress; switches on Kind live here only.
type LifecycleAddress struct {
	Kind AddressKind
	// Slot arm. Root is the slot's physical checkout and alone forms the key,
	// so a :0 primary (no numbered SlotIdentity) is addressed by its root.
	Slot SlotIdentity
	Root string
	// Team arm.
	Team TeamIdentity
}

// NewSlotAddress addresses a slot by its physical checkout root. A numbered
// slot's root must be its own checkout; a :0 primary passes a zero
// SlotIdentity.
func NewSlotAddress(slot SlotIdentity, physicalRoot string) (LifecycleAddress, error) {
	a := LifecycleAddress{Kind: AddressSlot, Slot: slot, Root: physicalRoot}
	return a, a.Validate()
}

// NewTeamAddress resolves a team folder inside an operations checkout. It is
// the IO seam of the team arm: the folder must hold a .hub file.
func NewTeamAddress(checkoutRoot, teamDir string) (LifecycleAddress, error) {
	if !workspaceAbsolute(checkoutRoot) {
		return LifecycleAddress{}, fmt.Errorf("team checkout root %q is not a clean absolute path", checkoutRoot)
	}
	if !teamDirShape(teamDir) {
		return LifecycleAddress{}, fmt.Errorf("team folder %q is not a clean relative path inside its checkout", teamDir)
	}
	scope, err := launcher.ResolveRepoScope(checkoutRoot)
	if err != nil {
		return LifecycleAddress{}, err
	}
	hub := filepath.Join(checkoutRoot, teamDir, ".hub")
	if info, err := os.Stat(hub); err != nil {
		return LifecycleAddress{}, fmt.Errorf("team folder %s has no .hub: %w", teamDir, err)
	} else if !info.Mode().IsRegular() {
		return LifecycleAddress{}, fmt.Errorf("team folder %s: .hub is not a regular file", teamDir)
	}
	a := LifecycleAddress{Kind: AddressTeam, Team: TeamIdentity{
		CheckoutRoot: checkoutRoot,
		TeamDir:      teamDir,
		Name:         filepath.Base(teamDir),
		Scope:        scope.Key,
	}}
	return a, a.Validate()
}

// Validate is a pure shape check: exactly one arm, matching Kind, with its
// resolved fields present and consistent.
func (a LifecycleAddress) Validate() error {
	switch a.Kind {
	case AddressSlot:
		if a.Team != (TeamIdentity{}) {
			return fmt.Errorf("slot address carries a team")
		}
		if !workspaceAbsolute(a.Root) {
			return fmt.Errorf("slot root %q is not a clean absolute path", a.Root)
		}
		if a.Slot != (SlotIdentity{}) {
			if err := a.Slot.validateLocation(); err != nil {
				return err
			}
			if a.Slot.WorktreeRoot != a.Root {
				return fmt.Errorf("slot root %q is not the slot's checkout %q", a.Root, a.Slot.WorktreeRoot)
			}
		}
		return nil
	case AddressTeam:
		if a.Root != "" || a.Slot != (SlotIdentity{}) {
			return fmt.Errorf("team address carries a slot")
		}
		t := a.Team
		if !workspaceAbsolute(t.CheckoutRoot) || !teamDirShape(t.TeamDir) {
			return fmt.Errorf("team address has an invalid location")
		}
		if t.Name != filepath.Base(t.TeamDir) {
			return fmt.Errorf("team name %q does not match its folder %q", t.Name, t.TeamDir)
		}
		if err := launcher.ValidateRepoScopeKey(t.Scope); err != nil {
			return fmt.Errorf("team scope unresolved: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unknown lifecycle address kind %q", a.Kind)
	}
}

// Key is the address's stable identity: "slot:<root>" or
// "team:<scope>/<TeamDir>". File names derived from it are digests, never the
// raw key.
func (a LifecycleAddress) Key() string {
	switch a.Kind {
	case AddressSlot:
		return slotLifecycleKey(a.Root)
	case AddressTeam:
		return "team:" + a.Team.Scope + "/" + a.Team.TeamDir
	default:
		return ""
	}
}

// slotLifecycleKey is the one definition of a slot's key, shared with #683's
// failure log (LifecycleFailureSlotKey).
func slotLifecycleKey(root string) string { return "slot:" + root }

func teamDirShape(dir string) bool {
	return dir != "" && dir != "." && filepath.IsLocal(dir) && filepath.Clean(dir) == dir
}
