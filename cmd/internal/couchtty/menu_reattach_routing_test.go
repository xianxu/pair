package couchtty

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// The pass routed through ReduceMenu (pair#206 plan, Task 7), and the
// generated-sequence invariants (Task 6 Step 0) that hold over every
// interleaving rather than the one each hand-written case samples.

// passMenu is a console-shaped menu: armed with the startup root, then given
// its first inventory, exactly as the console will do it.
func passMenu(t *testing.T, inventory []couchcore.ActionableThreadSummary) (MenuState, []MenuEffect) {
	t.Helper()
	state := NewMenuState(nil, menuAddress("couch-root"))
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventReattachArm, Address: menuAddress("couch-root")})
	return ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: inventory, Generation: 1})
}

func passInventory() []couchcore.ActionableThreadSummary {
	return []couchcore.ActionableThreadSummary{
		reattachRow("couch-root", couchcore.ThreadLive, 0),
		reattachRow("couch-a", couchcore.ThreadDetached, 1),
		reattachRow("couch-b", couchcore.ThreadDetached, 2),
		reattachRow("couch-live", couchcore.ThreadLive, 3),
		reattachRow("couch-parked", couchcore.ThreadParked, 4),
	}
}

// The first inventory seeds the pass AND starts its first attempt: the effect
// comes back out of ReduceMenu for the console to dispatch.
func TestReduceMenuSeedsAndStartsThePassFromTheFirstInventory(t *testing.T) {
	state, effects := passMenu(t, passInventory())
	if state.Reattach.Phase != ReattachRunning || state.Reattach.Loading != menuAddress("couch-a") {
		t.Fatalf("pass = %+v, want running with couch-a loading", state.Reattach)
	}
	if len(effects) != 1 || !effects[0].Background || effects[0].Args["warm-only"] != "true" {
		t.Fatalf("effects = %+v, want the first warm-only background attempt", effects)
	}
}

// A pass result goes to the pass and never to the operator's slot -- it must
// get past the in-flight match, which would otherwise drop it on the floor.
func TestReduceMenuRoutesAPassResultAroundTheOperatorsSlot(t *testing.T) {
	state, effects := passMenu(t, passInventory())
	operator := MenuOperationOrigin{Operation: "switch", Attempt: 99, Address: menuAddress("couch-live")}
	state.InFlight = operator

	result := MenuEvent{Kind: MenuEventOperationResult, Operation: "resume", Background: true,
		Attempt: effects[0].Attempt, Address: menuAddress("couch-a"), Success: true, ProjectionAfterGeneration: 1}
	state, next := ReduceMenu(state, result)

	if _, attached := state.Reattach.Attached[menuAddress("couch-a")]; !attached {
		t.Fatalf("the pass result was dropped: %+v", state.Reattach)
	}
	if state.InFlight != operator {
		t.Fatalf("InFlight = %+v, want the operator's operation untouched", state.InFlight)
	}
	// And the pass holds behind the operator's operation (cell 10).
	if len(next) != 0 {
		t.Fatalf("the pass advanced under an operator operation: %+v", next)
	}
}

// Cell 10's other half: when the operator's slot clears, a held pass resumes.
//
// The hold only exists if the operator's operation starts WHILE a pass attempt
// is still running: then that attempt's completion finds the slot busy and the
// pass must not start the next thread. An earlier version of this test started
// the operator's operation after the attempt finished, by which point the pass
// had already advanced -- so there was nothing held for the cleared slot to
// release, and the test asserted a resumption that could not happen.
func TestReduceMenuResumesAHeldPassWhenTheOperatorsSlotClears(t *testing.T) {
	state, effects := passMenu(t, passInventory()) // couch-a is reattaching
	// The operator starts an operation on a live row while couch-a is in flight.
	state.Frames[0].SelectedAddress = menuAddress("couch-live")
	state, started := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(started) != 1 || state.InFlight.Operation == "" {
		t.Fatalf("operator Enter = %+v (inflight %+v)", started, state.InFlight)
	}
	// couch-a finishes while the operator's operation holds the slot: the pass
	// HOLDS rather than starting couch-b.
	state, held := ReduceMenu(state, MenuEvent{Kind: MenuEventOperationResult, Operation: "resume", Background: true,
		Attempt: effects[0].Attempt, Address: menuAddress("couch-a"), Success: true})
	if len(held) != 0 || state.Reattach.Loading != (couchcore.ThreadAddress{}) {
		t.Fatalf("the pass advanced under the operator's operation: effects %+v, loading %v", held, state.Reattach.Loading)
	}
	// The operator's operation completes; the slot clears; the held pass resumes.
	_, resumed := ReduceMenu(state, correlatedMenuResult(state, MenuEvent{
		Operation: state.InFlight.Operation, Address: state.InFlight.Address, Success: true,
	}))
	if len(resumed) != 1 || !resumed[0].Background || resumed[0].Args["tag"] != "couch-b" {
		t.Fatalf("effects after the slot cleared = %+v, want the pass's next attempt on couch-b", resumed)
	}
}

// The cursor never lands on a pending row, in either direction.
func TestTheCursorSkipsPendingRows(t *testing.T) {
	state, _ := passMenu(t, passInventory()) // couch-a loading, couch-b queued
	for step := 0; step < 8; step++ {
		key := KeyDown
		if step >= 4 {
			key = KeyUp
		}
		state, _ = reduceKey(state, PanelKey{Kind: key})
		selected := state.CurrentFrame().SelectedAddress
		if view, owned := passViewOf(state.Reattach, selected); owned && view.Pending() {
			t.Fatalf("step %d: the cursor landed on pending %v", step, selected)
		}
	}
}

// A filter that matches only pending rows selects nothing, and Enter says so.
func TestAFilterMatchingOnlyPendingRowsSelectsNothing(t *testing.T) {
	inventory := passInventory()
	inventory[1].Name, inventory[2].Name = "pending-one", "pending-two"
	state, _ := passMenu(t, inventory)
	for _, r := range "pending" {
		state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: r})
	}
	if got := state.CurrentFrame().SelectedAddress; got != (couchcore.ThreadAddress{}) {
		t.Fatalf("selection = %v, want none: every match is pending", got)
	}
	state, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 || state.Notice.Text != "no selection" {
		t.Fatalf("Enter with only pending matches: effects %+v, notice %q", effects, state.Notice.Text)
	}
}

// A click on a pending row lands nowhere.
func TestAClickOnAPendingRowDoesNothing(t *testing.T) {
	state, _ := passMenu(t, passInventory())
	for _, tag := range []string{"couch-a", "couch-b"} {
		if _, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventMouseSwitch, Address: menuAddress(tag)}); len(effects) != 0 {
			t.Fatalf("a click on pending %s dispatched %+v", tag, effects)
		}
	}
}

// An attached row switches: the view makes it live while the inventory lags,
// so Enter does not offer a resume couchcore would refuse as occupied.
func TestAnAttachedRowSwitchesWhileTheInventoryLags(t *testing.T) {
	state, effects := passMenu(t, passInventory())
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventOperationResult, Operation: "resume", Background: true,
		Attempt: effects[0].Attempt, Address: menuAddress("couch-a"), Success: true, ProjectionAfterGeneration: 1})
	state.Frames[0].SelectedAddress = menuAddress("couch-a") // the inventory still says detached
	_, dispatched := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(dispatched) != 1 || dispatched[0].Operation != "switch" {
		t.Fatalf("Enter on the just-attached row = %+v, want a switch", dispatched)
	}
}

// Enter on a failed row is an ordinary resume, and clears the mark (cell 9).
func TestEnterOnAFailedRowResumesAndClearsTheMark(t *testing.T) {
	state, effects := passMenu(t, passInventory())
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventOperationResult, Operation: "resume", Background: true,
		Attempt: effects[0].Attempt, Address: menuAddress("couch-a"), Diagnostic: couchcore.ResumePathMissing})
	if _, failed := state.Reattach.Failed[menuAddress("couch-a")]; !failed {
		t.Fatalf("setup: %+v", state.Reattach)
	}
	state.InFlight = MenuOperationOrigin{} // the pass's next attempt is its own, not the operator's
	state.Frames[0].SelectedAddress = menuAddress("couch-a")
	state, dispatched := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(dispatched) != 1 || dispatched[0].Operation != "resume" || dispatched[0].Background {
		t.Fatalf("Enter on a failed row = %+v, want an ordinary operator resume", dispatched)
	}
	if _, still := state.Reattach.Failed[menuAddress("couch-a")]; still {
		t.Fatal("resuming by hand did not clear the failure mark")
	}
}

// The invariants, over generated event sequences. A hand-written case samples
// one interleaving; these must hold after EVERY step of every sequence.
func TestReattachPassInvariantsHoldOverGeneratedSequences(t *testing.T) {
	addresses := []couchcore.ThreadAddress{}
	for _, row := range passInventory() {
		addresses = append(addresses, row.Address)
	}
	for seed := int64(1); seed <= 60; seed++ {
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			rng := rand.New(rand.NewSource(seed))
			state, _ := passMenu(t, passInventory())
			seededLen := len(state.Reattach.Queue) + 1 // the first attempt already popped one
			generation := uint64(1)
			for step := 0; step < 120; step++ {
				before := state
				var event MenuEvent
				switch rng.Intn(11) {
				case 0:
					generation++
					event = MenuEvent{Kind: MenuEventInventory, Inventory: passInventory(), Generation: generation}
				case 1:
					event = MenuEvent{Kind: MenuEventInventory, Error: "refresh failed", Generation: generation}
				case 2, 3:
					if state.Reattach.Loading == (couchcore.ThreadAddress{}) {
						continue
					}
					event = MenuEvent{Kind: MenuEventOperationResult, Operation: "resume", Background: true,
						Attempt: state.Reattach.LoadingAttempt, Address: state.Reattach.Loading, ProjectionAfterGeneration: generation}
					switch rng.Intn(3) {
					case 0:
						event.Success = true
					case 1:
						event.Diagnostic = couchcore.ResumeNotDetached
					default:
						event.Diagnostic = couchcore.ResumePathMissing
					}
				case 4:
					if state.InFlight.Operation == "" {
						continue
					}
					event = correlatedMenuResult(state, MenuEvent{Operation: state.InFlight.Operation, Address: state.InFlight.Address, Success: true})
				case 5:
					event = MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyDown}}
				case 6:
					event = MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyUp}}
				case 7:
					event = MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyEnter}}
				case 8:
					event = MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyTab}}
				case 9:
					event = MenuEvent{Kind: MenuEventMouseSwitch, Address: addresses[rng.Intn(len(addresses))]}
				default:
					event = MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyEscape}}
				}
				var effects []MenuEffect
				state, effects = ReduceMenu(state, event)
				assertReattachInvariants(t, before, state, effects, seededLen, fmt.Sprintf("seed %d step %d event %d", seed, step, event.Kind))
			}
		})
	}
}

func assertReattachInvariants(t *testing.T, before, after MenuState, effects []MenuEffect, seededLen int, where string) {
	t.Helper()
	pass := after.Reattach
	root := menuAddress("couch-root")
	if len(pass.Queue) > seededLen {
		t.Fatalf("%s: the queue grew past its seeded length: %v", where, pass.Queue)
	}
	seen := map[couchcore.ThreadAddress]string{}
	claim := func(address couchcore.ThreadAddress, where2 string) {
		if prior, dup := seen[address]; dup {
			t.Fatalf("%s: %v is both %s and %s", where, address, prior, where2)
		}
		seen[address] = where2
	}
	if pass.Loading != (couchcore.ThreadAddress{}) {
		claim(pass.Loading, "loading")
	}
	for _, address := range pass.Queue {
		if address == root {
			t.Fatalf("%s: the root was queued", where)
		}
		claim(address, "queued")
	}
	for address := range pass.Failed {
		claim(address, "failed")
	}
	for address := range pass.Attached {
		claim(address, "attached")
	}
	// A pending row is never the selection.
	if selected := after.Frames[0].SelectedAddress; selected != (couchcore.ThreadAddress{}) {
		if view, owned := passViewOf(pass, selected); owned && view.Pending() {
			t.Fatalf("%s: pending %v is the selection", where, selected)
		}
	}
	for _, effect := range effects {
		if effect.Background {
			// The pass never emits while the operator holds the slot.
			if after.InFlight.Operation != "" {
				t.Fatalf("%s: a pass attempt emitted while the operator's %q was in flight", where, after.InFlight.Operation)
			}
			continue
		}
		// No operator Enter, click or Tab ever acts on a row that was pending.
		if tag := effect.Args["tag"]; tag != "" {
			address := couchcore.ThreadAddress{RepoScope: effect.Args["repo-scope"], Tag: couchcore.ThreadTag(tag)}
			if view, owned := passViewOf(before.Reattach, address); owned && view.Pending() {
				t.Fatalf("%s: the operator dispatched %q on pending %v", where, effect.Operation, address)
			}
		}
	}
}

// A successful leave ends the console, so it ends the pass. Without this, the
// leave's result clears the operator's slot, the pass advances, and one more
// reattach is enqueued only for Stop to cancel it -- the outcome cell 10 exists
// to prevent.
func TestASuccessfulLeaveEndsThePass(t *testing.T) {
	state, effects := passMenu(t, passInventory()) // couch-a reattaching, couch-b queued
	// The operator leaves while couch-a is in flight; the pass holds.
	state, started := dispatchMenuOperation(state, MenuEffect{Operation: "leave"}, couchcore.ThreadAddress{})
	if len(started) != 1 {
		t.Fatalf("leave did not dispatch: %+v", started)
	}
	state, held := ReduceMenu(state, MenuEvent{Kind: MenuEventOperationResult, Operation: "resume", Background: true,
		Attempt: effects[0].Attempt, Address: menuAddress("couch-a"), Success: true})
	if len(held) != 0 {
		t.Fatalf("the pass advanced under the leave: %+v", held)
	}
	state, after := ReduceMenu(state, correlatedMenuResult(state, MenuEvent{Operation: "leave", Success: true}))
	if len(after) != 0 {
		t.Fatalf("a successful leave was followed by %+v; the console is ending", after)
	}
	if state.Reattach.Phase != ReattachDone {
		t.Fatalf("pass phase = %d after a leave, want Done", state.Reattach.Phase)
	}
}

// Cell 11 through the reducer: an inventory newer than an attach is
// authoritative for that thread again. The pure test drives expireAttached
// directly, so the M2 sweep's mutant that dropped the reducer's call to it
// survived. Without the call the pass never lets go of the row, and a thread
// whose pane exited would read live forever.
func TestANewerInventoryTakesBackARowThePassAttached(t *testing.T) {
	inventory := []couchcore.ActionableThreadSummary{
		reattachRow("couch-root", couchcore.ThreadLive, 0),
		reattachRow("couch-a", couchcore.ThreadDetached, 1),
	}
	state, effects := passMenu(t, inventory)
	if len(effects) != 1 {
		t.Fatalf("setup: %d effects, want the one attempt for couch-a", len(effects))
	}
	live := func(when string) bool {
		t.Helper()
		row, ok := menuThread(state, menuAddress("couch-a"))
		if !ok {
			t.Fatalf("%s: couch-a is not in the inventory", when)
		}
		return row.Live()
	}
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventOperationResult, Operation: "resume", Background: true,
		Attempt: effects[0].Attempt, Address: menuAddress("couch-a"), Success: true, ProjectionAfterGeneration: 50})
	if !live("after the attach") {
		t.Fatal("the pass does not read couch-a live after attaching it")
	}
	// Not newer than the attach: the lagging inventory still says detached, and
	// the pass still owns the row.
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: inventory, Generation: 50})
	if !live("at the attach's own generation") {
		t.Fatal("an inventory from the attach's own generation took the row back")
	}
	// Newer: authoritative again, whatever it says.
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: inventory, Generation: 51})
	if live("after a newer inventory") {
		t.Fatal("a newer inventory saying detached left couch-a live: the pass never let go of the row")
	}
}

// Cell 2 through the reducer: a failed first refresh leaves the pass armed.
// Seeding from it would read "no threads" and end the pass before it ever saw a
// real inventory. The M2 sweep's cell-2 mutant did exactly that and survived,
// because nothing drove an error inventory into an armed pass.
func TestAFailedFirstInventoryLeavesThePassArmed(t *testing.T) {
	root := menuAddress("couch-root")
	state, _ := ReduceMenu(NewMenuState(nil, root), MenuEvent{Kind: MenuEventReattachArm, Address: root})
	state, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Error: "refresh failed", Generation: 1})
	if state.Reattach.Phase != ReattachArmed || len(state.Reattach.Queue) != 0 || len(effects) != 0 {
		t.Fatalf("after a failed inventory: phase %v, queue %v, effects %v; want still armed and nothing started",
			state.Reattach.Phase, state.Reattach.Queue, effects)
	}
	inventory := []couchcore.ActionableThreadSummary{
		reattachRow("couch-root", couchcore.ThreadLive, 0),
		reattachRow("couch-a", couchcore.ThreadDetached, 1),
	}
	state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: inventory, Generation: 2})
	if state.Reattach.Loading != menuAddress("couch-a") || len(effects) != 1 {
		t.Fatalf("after the first good inventory: loading %v, effects %v; want the pass seeded and couch-a started",
			state.Reattach.Loading, effects)
	}
}
