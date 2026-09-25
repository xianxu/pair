package couchcore

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func (c *Couch) resolveManagedStart(ctx context.Context, args StartArgs) (StartResolution, error) {
	action := args.Action
	if action == "" {
		action = StartOpen
	}
	if action != StartOpen && action != StartCreate && action != StartFresh {
		return StartResolution{}, fmt.Errorf("unknown start action %q", action)
	}
	original := args.WorkingDir()
	if original == "" {
		original = "."
	}
	target, id, err := c.resolveSlotInput(ctx, original)
	if err != nil {
		return StartResolution{}, err
	}
	if _, explicit, _ := ParseWorkspaceReference(original); explicit && action == StartCreate {
		action = StartOpen
	}
	if id.Kind != "primary" && id.Kind != "slot" {
		return c.resolveOrdinaryStartResolution(ctx, args)
	}
	primary := id.PrimaryRoot
	var reuseNotices []StartReuseNotice
	reuse := false
	if action == StartCreate {
		repository, e := c.Slots.Discover(ctx, primary)
		if e != nil {
			return StartResolution{}, e
		}
		snapshot, e := c.Threads.PreviewSnapshot()
		if e != nil {
			return StartResolution{}, e
		}
		// Start fills the lowest number, :0 included, holding no thread (#331,
		// #332). Archived threads have left the snapshot, so their numbers are
		// free; a threadless checkout is reused as-is with a fresh conversation.
		numbers := slotRepositoryNumbers(repository)
		for _, address := range snapshot.Unreadable {
			if _, ok := numbers[address.RepoScope]; ok {
				return StartResolution{}, fmt.Errorf("repository has an unreadable conversation; recover it before creating another slot")
			}
		}
		for _, slot := range snapshot.Slots {
			if slot.Err != nil && slot.Identity.RepoIdentity == repository.Identity.RepoIdentity {
				return StartResolution{}, fmt.Errorf("repository slot %d needs attention before creating another slot: %w", slot.Identity.Number, slot.Err)
			}
		}
		occupied := map[int]bool{}
		for _, record := range snapshot.Records {
			if number, ok := numbers[record.Address.RepoScope]; ok {
				occupied[number] = true
			}
		}
		allocation, e := SelectStartSlot(repository.Slots, occupied)
		if e != nil {
			return StartResolution{}, e
		}
		if allocation.Number != 0 {
			reuseNotices, e = c.reuseNoticesInRepository(ctx, repository)
			if e != nil {
				return StartResolution{}, e
			}
		}
		target = nil
		if allocation.Number != 0 {
			slot := conventionalSlot(primary, allocation.Number)
			slot.RepoIdentity = repository.Identity.RepoIdentity
			target = &ThreadTarget{Kind: ThreadTargetSlot, Slot: slot}
			if allocation.Exists {
				reuse = true
			}
		}
	}
	path := primary
	if target != nil {
		path = target.Slot.WorktreeRoot
	}
	var resolution StartResolution
	if target == nil && (strings.ContainsAny(original, `/\`) || original == ".") {
		resolution, err = c.resolveOrdinaryStartResolution(ctx, args)
	} else {
		resolution, err = c.resolveStartProfile(args, path, Worktree(path), id.RepoIdentity, primary)
	}
	if err != nil {
		return StartResolution{}, err
	}
	resolution.OriginalInput, resolution.Action = original, action
	resolution.ReuseSlot, resolution.ReuseNotices = reuse, reuseNotices
	if target != nil {
		resolution.Target = *target
	} else if action == StartFresh {
		return StartResolution{}, errors.New("fresh action requires an existing numbered slot")
	}
	resolution.Fingerprint = fingerprintStartResolution(resolution)
	return resolution, nil
}

// slotRepositoryNumbers maps each checkout's repo scope to its slot number,
// the primary as 0, so a thread record names the number it occupies.
func slotRepositoryNumbers(repository SlotRepository) map[string]int {
	numbers := map[string]int{}
	add := func(path string, number int) {
		if scope, err := launcher.ResolveRepoScope(path); err == nil {
			numbers[scope.Key] = number
		}
	}
	add(repository.Identity.PrimaryRoot, 0)
	for _, slot := range repository.Slots {
		add(slot.Identity.WorktreeRoot, slot.Identity.Number)
	}
	return numbers
}

// repositoryRows are the switcher rows belonging to repository.
func (c *Couch) repositoryRows(ctx context.Context, repository SlotRepository) ([]ActionableThreadSummary, error) {
	rows, err := c.ActionableThreadInventoryContext(ctx, nil)
	if err != nil {
		return nil, err
	}
	numbers := slotRepositoryNumbers(repository)
	var mine []ActionableThreadSummary
	for _, row := range rows {
		_, scoped := numbers[row.Address.RepoScope]
		if scoped || (row.Target.Kind == ThreadTargetSlot && row.Target.Slot.RepoIdentity == repository.Identity.RepoIdentity) {
			mine = append(mine, row)
		}
	}
	return mine, nil
}

// reuseNoticesInRepository labels parked threads and lost-binding slots. Since
// #332 they no longer block a new slot; the start preview names them instead,
// so existing work stays in view without stopping the operator.
func (c *Couch) reuseNoticesInRepository(ctx context.Context, repository SlotRepository) ([]StartReuseNotice, error) {
	rows, err := c.repositoryRows(ctx, repository)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(rows, func(i, j int) bool {
		left, right := rows[i], rows[j]
		leftNumber, rightNumber := 0, 0
		if left.Target.Kind == ThreadTargetSlot {
			leftNumber = left.Target.Slot.Number
		}
		if right.Target.Kind == ThreadTargetSlot {
			rightNumber = right.Target.Slot.Number
		}
		return leftNumber < rightNumber
	})
	notices := []StartReuseNotice{}
	for _, row := range rows {
		if row.State == ThreadParked {
			notices = append(notices, StartReuseNotice{Kind: StartReuseNoticeParked, Label: row.Label(), Slot: rowSlotNumber(row)})
		}
		if row.Target.Kind == ThreadTargetSlot && row.State == ThreadUnusable && row.Reason == ReasonBindingLost {
			notices = append(notices, StartReuseNotice{Kind: StartReuseNoticeLost, Label: row.Label(), Slot: row.Target.Slot.Number})
		}
	}
	sortReuseNotices(notices)
	return notices, nil
}

func sortReuseNotices(notices []StartReuseNotice) {
	sort.SliceStable(notices, func(i, j int) bool {
		if notices[i].Slot != notices[j].Slot {
			return notices[i].Slot < notices[j].Slot
		}
		return notices[i].Kind < notices[j].Kind
	})
}

func rowSlotNumber(row ActionableThreadSummary) int {
	if row.Target.Kind == ThreadTargetSlot {
		return row.Target.Slot.Number
	}
	return 0
}

// checkSlotCreation refuses a new slot only while a thread's ownership is
// unknown; parked work is named in the preview instead (#332).
func (c *Couch) checkSlotCreation(ctx context.Context, repository SlotRepository) error {
	rows, err := c.repositoryRows(ctx, repository)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.State == ThreadUnusable && (row.Reason == ReasonUnreadable || row.Reason == ReasonUnknown) {
			return fmt.Errorf("repository ownership needs attention at %s before creating another slot", row.Label())
		}
	}
	return nil
}

func (c *Couch) spawnManagedResolution(ctx context.Context, resolution StartResolution) (StartResult, error) {
	if resolution.Target.Kind != ThreadTargetSlot {
		return StartResult{}, errors.New("managed launch requires a slot target")
	}
	slot := resolution.Target.Slot
	switch resolution.Action {
	case StartFresh:
		return c.StartFreshSlot(ctx, slot.WorktreeRoot, resolution.RequestedAgent)
	case StartOpen:
		return c.OpenSlot(ctx, slot.WorktreeRoot, resolution.RequestedAgent)
	case StartCreate:
		if resolution.ReuseSlot {
			return c.startFreshSlot(ctx, slot.WorktreeRoot, resolution.RequestedAgent, true, &resolution)
		}
	default:
		return StartResult{}, fmt.Errorf("invalid slot start action %q", resolution.Action)
	}
	repository, err := c.Slots.Discover(ctx, slot.PrimaryRoot)
	if err != nil {
		return StartResult{}, err
	}
	for _, candidate := range repository.Slots {
		if candidate.Identity.Number == slot.Number {
			return StartResult{}, ErrStartResolutionChanged
		}
	}
	if err := c.Threads.EnrollSlotRepository(ctx, repository); err != nil {
		return StartResult{}, err
	}
	if err := c.checkSlotCreation(ctx, repository); err != nil {
		return StartResult{}, err
	}
	if c.Workspaces == nil {
		return StartResult{}, errors.New("workspace readiness unavailable")
	}
	result, err := c.Workspaces.Ensure(ctx, ProvisionRequest{Path: slot.PrimaryRoot, Slot: slot.Number, Progress: c.WorkspaceProgress})
	if err != nil {
		return StartResult{}, err
	}
	if result.SchemaVersion != 1 || filepath.Clean(result.Path) != slot.WorktreeRoot || result.Address != (WorkspaceReference{Repo: slot.Repo, Number: slot.Number}).String() {
		return StartResult{}, errors.New("created workspace does not match accepted slot")
	}
	// A competing creator may have won after preview. Do not adopt its result as
	// our new slot or silently increment the number.
	if result.Disposition != "created" {
		return StartResult{}, fmt.Errorf("slot appeared during creation; open %s explicitly", result.Address)
	}
	repository, err = c.Slots.Discover(ctx, slot.PrimaryRoot)
	if err != nil {
		return StartResult{}, err
	}
	if err = c.Threads.EnrollSlotRepository(ctx, repository); err != nil {
		return StartResult{}, err
	}
	if err = c.checkSlotCreation(ctx, repository); err != nil {
		return StartResult{}, err
	}
	rows, err := c.ActionableThreadInventoryContext(ctx, nil)
	if err != nil {
		return StartResult{}, err
	}
	record, handle, err := c.spawnResolved(ctx, resolution, rows)
	return StartResult{Record: record, Handle: handle}, err
}

// Recheck profile, physical identity and repository parked work after setup;
// the accepted number is fixed even though discovery now sees the new directory.
func (c *Couch) revalidateCreatedSlot(ctx context.Context, accepted StartResolution) error {
	slot := accepted.Target.Slot
	identity, err := c.slotWorkspace(ctx, slot.WorktreeRoot)
	if err != nil {
		return err
	}
	currentSlot, err := SlotIdentityFromWorkspace(identity)
	if err != nil {
		return err
	}
	if currentSlot != slot {
		return ErrStartResolutionChanged
	}
	current, err := c.resolveStartProfile(StartArgs{Stack: accepted.RequestedAgent, Issue: accepted.Issue}, slot.WorktreeRoot, Worktree(slot.WorktreeRoot), slot.RepoIdentity, slot.PrimaryRoot)
	if err != nil {
		return err
	}
	current.Action, current.OriginalInput, current.Target = accepted.Action, accepted.OriginalInput, accepted.Target
	current.ReuseSlot = accepted.ReuseSlot
	if fingerprintStartResolution(current) != accepted.Fingerprint {
		return ErrStartResolutionChanged
	}
	repository, err := c.Slots.Discover(ctx, slot.PrimaryRoot)
	if err != nil {
		return err
	}
	return c.checkSlotCreation(ctx, repository)
}

func (c *Couch) enrollPrimaryResolution(ctx context.Context, resolution StartResolution) error {
	if c.Slots == nil || resolution.Action == "" || resolution.Target.Kind == ThreadTargetSlot {
		return nil
	}
	repository, err := c.Slots.Discover(ctx, string(resolution.Worktree))
	if err != nil {
		return err
	}
	if repository.Identity.RepoIdentity != resolution.RepoIdentity {
		return ErrStartResolutionChanged
	}
	return c.Threads.EnrollSlotRepository(ctx, repository)
}
