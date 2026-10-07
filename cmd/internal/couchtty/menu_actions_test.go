package couchtty

import (
	"maps"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

// menuRowShape is one row the projection can produce, described by the facts
// the Spec table is written over, so the expectation can be stated without
// calling into production.
type menuRowShape struct {
	name    string
	slot    bool
	state   couchcore.ActionableThreadState
	reason  couchcore.ThreadReason
	phase   checkpoint.Phase // "" = no retained request
	recover bool             // unusable :0 with Recovery.Recover
	row     couchcore.ActionableThreadSummary
}

// everyMenuRowShape is the DERIVED row domain every per-row sweep iterates:
// kinds {:0, :1} x AllThreadStates() (minus archived) x AllThreadReasons() ∪ ""
// (only the combinations the projection produces: a reason exactly when
// unusable) x AllPhases() ∪ none, plus Recovery.Recover on unusable :0 rows. A
// new state, reason or phase lands here without anyone listing it.
func everyMenuRowShape(t *testing.T) []menuRowShape {
	t.Helper()
	scope, err := launcher.ResolveRepoScope("/w/xianxu.dev")
	if err != nil {
		t.Fatal(err)
	}
	// "" is no retained request; appended, not written as a phase literal,
	// which TestPhaseListIsWrittenOnlyInAllPhases forbids outside AllPhases.
	phases := append(make([]checkpoint.Phase, 0, 5), "")
	phases = append(phases, checkpoint.AllPhases()...)
	var shapes []menuRowShape
	for _, slot := range []bool{false, true} {
		for _, state := range couchcore.AllThreadStates() {
			if state == couchcore.ThreadArchived {
				continue
			}
			for _, reason := range append(couchcore.AllThreadReasons(), "") {
				if (state == couchcore.ThreadUnusable) != (reason != "") {
					continue
				}
				for _, phase := range phases {
					recovers := []bool{false}
					if !slot && state == couchcore.ThreadUnusable {
						recovers = append(recovers, true)
					}
					for _, recover := range recovers {
						var row couchcore.ActionableThreadSummary
						if slot {
							row = menuSlotRow(1, "couch-slot")
						} else {
							row = couchcore.ActionableThreadSummary{
								Address:      couchcore.ThreadAddress{RepoScope: scope.Key, Tag: "couch-primary"},
								StartingPath: "/w/xianxu.dev", WorkingPath: "/w/xianxu.dev",
							}
						}
						row.State, row.Reason = state, reason
						if phase != "" {
							row.Continuation = &couchcore.ContinuationStatus{RequestID: "request", Phase: phase}
						}
						if recover {
							row.Recovery = &couchcore.RecoveryDecision{Recover: true, Archive: true}
						}
						kind := ":0"
						if slot {
							kind = ":1"
						}
						shapes = append(shapes, menuRowShape{
							name: kind + "/" + string(state) + "/" + string(reason) + "/" + string(phase) + map[bool]string{true: "/recover"}[recover],
							slot: slot, state: state, reason: reason, phase: phase, recover: recover, row: row,
						})
					}
				}
			}
		}
	}
	return shapes
}

// expectedRowActions is the Spec table, written out as its own statement.
func expectedRowActions(s menuRowShape) []string {
	unfinished := s.phase != "" && s.phase != checkpoint.Complete
	switch s.state {
	case couchcore.ThreadBusy:
		return nil
	case couchcore.ThreadLive:
		switch {
		case unfinished && s.phase == checkpoint.Running:
			return []string{"retry-continuation"}
		case unfinished && s.phase == checkpoint.Pending:
			return nil
		case unfinished && s.phase == checkpoint.Failed:
			// relaunch and switch-agent are what a failed request refuses.
			if s.slot {
				return []string{"detach", "retry-continuation", "dismiss-continuation", "park"}
			}
			return []string{"detach", "retry-continuation", "dismiss-continuation", "park", "alias", "add-slot"}
		}
		if s.slot {
			return []string{"detach", "relaunch", "park", "switch-agent"}
		}
		return []string{"detach", "relaunch", "park", "switch-agent", "alias", "add-slot"}
	case couchcore.ThreadParked, couchcore.ThreadDetached:
		return []string{"recover", "resume", "reboot"}
	case couchcore.ThreadUnusable:
		switch {
		case s.reason == couchcore.ReasonUnknown:
			return nil
		// An orphaned server's agent may still be writing: neither resume nor
		// reboot is safe; only the confirmed reap (#399).
		case s.reason == couchcore.ReasonOrphanedServer:
			return []string{"recover", "reap"}
		case s.reason == couchcore.ReasonPathMissing && s.slot:
			return nil
		case s.reason == couchcore.ReasonPathMissing:
			return []string{"recover", "reboot"}
		// A parked slot whose conversation cannot be resolved (its agent
		// never took a turn) has nothing to resume (pair#367 smoke test).
		case s.slot && s.reason == couchcore.ReasonBindingLost && !unfinished:
			return []string{"recover", "reboot"}
		case s.slot || s.recover || unfinished:
			return []string{"recover", "resume", "reboot"}
		}
		return []string{"recover", "reboot"}
	}
	return nil
}

func TestRowActionTableMatchesTheSpec(t *testing.T) {
	shapes := everyMenuRowShape(t)
	if len(shapes) < 50 {
		t.Fatalf("derived domain has only %d shapes", len(shapes))
	}
	for _, s := range shapes {
		got, want := menuActionItems(s.row), expectedRowActions(s)
		if !slices.Equal(got, want) {
			t.Errorf("%s: offers %v, Spec says %v", s.name, got, want)
		}
		// Resume's refusal advice (couchcore.withRebootAdvice) says
		// "Tab → reboot": every row that offers resume must offer reboot.
		if slices.Contains(got, "resume") && !slices.Contains(got, "reboot") {
			t.Errorf("%s: offers resume without reboot, so resume's advice names an action the row lacks", s.name)
		}
	}
}

// A busy row's status and its Enter notice come from the table's busy phase,
// one per-row authority, and both say what is actually happening.
func TestBusyRowSaysStartingElsewhere(t *testing.T) {
	busy := couchcore.ActionableThreadSummary{
		Address: menuAddress("couch-one"), WorkingPath: "/repo/one", State: couchcore.ThreadBusy,
	}
	if got := rootStateText(busy, time.Now()); got != "starting elsewhere" {
		t.Fatalf("busy status = %q, want %q", got, "starting elsewhere")
	}
	state := NewMenuState([]couchcore.ActionableThreadSummary{busy}, couchcore.ThreadAddress{})
	state.InventoryReady = true
	got, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 || !strings.Contains(got.Notice.Text, "starting elsewhere") {
		t.Fatalf("Enter on busy: effects %v, notice %q", effects, got.Notice.Text)
	}
}

// Enter is the table's first action on rows that are not live: resume wherever
// it is offered, switch on a live row, and on any other row a notice that says
// why and names the action the row does offer.
func TestEnterResumesEveryRowThatOffersResume(t *testing.T) {
	for _, s := range everyMenuRowShape(t) {
		state := NewMenuState([]couchcore.ActionableThreadSummary{s.row}, s.row.Address)
		state.InventoryReady = true
		next, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
		items := menuActionItems(s.row)
		switch {
		case s.row.Live():
			if len(effects) != 1 || effects[0].Operation != "switch" {
				t.Errorf("%s: Enter on a live row dispatched %+v, want switch", s.name, effects)
			}
		case slices.Contains(items, "resume"):
			if len(effects) != 1 || effects[0].Operation != "resume" {
				t.Errorf("%s: Enter on a row offering resume dispatched %+v", s.name, effects)
			}
		default:
			if len(effects) != 0 {
				t.Errorf("%s: Enter dispatched %+v on a row that offers no resume", s.name, effects)
			}
			wantsReboot := slices.Contains(items, "reboot")
			if strings.Contains(next.Notice.Text, "Tab → reboot") != wantsReboot {
				t.Errorf("%s: notice %q, reboot offered %v", s.name, next.Notice.Text, wantsReboot)
			}
			if next.Notice.Text == "" {
				t.Errorf("%s: Enter said nothing", s.name)
			}
		}
	}
}

// A slot row is addressed by its host checkout -- its record may be missing or
// replaced under it -- and its in-flight operation is keyed by row. A :0 row is
// addressed by its exact tag.
func TestSlotRowResumeAndRebootSendThePath(t *testing.T) {
	slot := menuSlotRow(1, "couch-slot")
	slot.State, slot.Reason = couchcore.ThreadParked, ""
	primary := couchcore.ActionableThreadSummary{Address: menuAddress("couch-primary"), WorkingPath: "/w/p", State: couchcore.ThreadParked}
	for _, operation := range []string{"resume", "reboot"} {
		t.Run(operation+"/slot", func(t *testing.T) {
			next, effects := dispatchMenuRow(NewMenuState([]couchcore.ActionableThreadSummary{slot}, couchcore.ThreadAddress{}), operation, slot)
			if len(effects) != 1 || effects[0].Args["path"] != slot.Target.Slot.WorktreeRoot || effects[0].Args["tag"] != "" {
				t.Fatalf("effects %+v", effects)
			}
			if next.InFlight.RowKey != slot.RowKey {
				t.Fatalf("in flight %+v not keyed by row", next.InFlight)
			}
			if _, warm := effects[0].Args["warm-only"]; warm {
				t.Fatalf("a slot resume must go through the route, not warm-only: %+v", effects[0].Args)
			}
		})
		t.Run(operation+"/primary", func(t *testing.T) {
			_, effects := dispatchMenuRow(NewMenuState([]couchcore.ActionableThreadSummary{primary}, primary.Address), operation, primary)
			if len(effects) != 1 || effects[0].Args["repo-scope"] != "scope" || effects[0].Args["tag"] != "couch-primary" || effects[0].Args["path"] != "" {
				t.Fatalf("effects %+v", effects)
			}
		})
	}
}

// The switcher's resume and reboot effects carry exactly
// couchcore.ActorOperationArgs for every row shape, including warm-only on a
// detached :0 (it may only reattach, never cold-start), whether dispatched by
// row or by address (pair#367: the socket's admission reads the same mapping).
func TestSwitcherActorEffectsCarryActorOperationArgs(t *testing.T) {
	slot := menuSlotRow(1, "couch-slot")
	slot.State, slot.Reason = couchcore.ThreadDetached, ""
	detached := couchcore.ActionableThreadSummary{Address: menuAddress("couch-primary"), WorkingPath: "/w/p", State: couchcore.ThreadDetached}
	parked := detached
	parked.State = couchcore.ThreadParked
	slotScope, err := launcher.ResolveRepoScope(slot.Target.Slot.WorktreeRoot)
	if err != nil {
		t.Fatal(err)
	}
	literal := map[string]map[string]string{
		"slot":     {"path": slot.Target.Slot.WorktreeRoot, "repo-scope": slotScope.Key},
		"detached": {"repo-scope": "scope", "tag": "couch-primary", "warm-only": "true"},
		"parked":   {"repo-scope": "scope", "tag": "couch-primary"},
	}
	for name, row := range map[string]couchcore.ActionableThreadSummary{"slot": slot, "detached": detached, "parked": parked} {
		state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
		_, byRow := dispatchMenuRow(state, "resume", row)
		if len(byRow) != 1 || !maps.Equal(byRow[0].Args, literal[name]) || !maps.Equal(byRow[0].Args, couchcore.ActorOperationArgs(row, "resume")) {
			t.Errorf("%s resume by row: %+v, want %v", name, byRow, literal[name])
		}
		if row.Target.Kind != couchcore.ThreadTargetSlot {
			_, byAddress := dispatchThreadOperation(state, "resume", row.Address)
			if len(byAddress) != 1 || !maps.Equal(byAddress[0].Args, literal[name]) {
				t.Errorf("%s resume by address: %+v, want %v", name, byAddress, literal[name])
			}
		}
		_, reboot := dispatchMenuRow(state, "reboot", row)
		want := maps.Clone(literal[name])
		delete(want, "warm-only")
		if len(reboot) != 1 || !maps.Equal(reboot[0].Args, want) || !maps.Equal(reboot[0].Args, couchcore.ActorOperationArgs(row, "reboot")) {
			t.Errorf("%s reboot: %+v, want %v", name, reboot, want)
		}
	}
}

func openRebootConfirmation(t *testing.T, row couchcore.ActionableThreadSummary) MenuState {
	t.Helper()
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state.InventoryReady = true
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	if state.CurrentFrame().Kind != MenuFrameActions {
		t.Fatalf("Tab did not open actions: %+v, notice %q", state.CurrentFrame(), state.Notice.Text)
	}
	state.Frames[len(state.Frames)-1].SelectedItem = "reboot"
	state, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 || state.CurrentFrame().Kind != MenuFrameConfirmation || state.CurrentFrame().Action != "reboot" {
		t.Fatalf("reboot did not confirm: %+v (effects %+v)", state.CurrentFrame(), effects)
	}
	return state
}

func TestRebootConfirmationNamesItsCost(t *testing.T) {
	base := couchcore.ActionableThreadSummary{Address: menuAddress("couch-primary"), WorkingPath: "/w/p"}
	for _, tc := range []struct {
		state  couchcore.ActionableThreadState
		reason couchcore.ThreadReason
		want   string
	}{
		{couchcore.ThreadParked, "", " — archives this conversation, starts a fresh agent"},
		// "may survive": #274's measurement, quiesce reaps by SIGHUP.
		{couchcore.ThreadDetached, "", " — archives this conversation, starts a fresh agent, stops its session; its running agent may survive"},
		{couchcore.ThreadUnusable, couchcore.ReasonPathMissing, " — checkout missing: archives the record only; restore the checkout to start here again"},
	} {
		row := base
		row.State, row.Reason = tc.state, tc.reason
		state := openRebootConfirmation(t, row)
		items := confirmationMenuItems(state, state.CurrentFrame())
		want := "reboot " + row.Label() + tc.want
		if len(items) != 2 || items[1] != want {
			t.Errorf("%s/%s: items %q, want %q", tc.state, tc.reason, items, want)
		}
	}
}

// One rule for every confirmation: it survives only while its row still
// offers its action (or that action is the one in flight).
func TestConfirmationRechecksTheOfferForEveryAction(t *testing.T) {
	row := couchcore.ActionableThreadSummary{Address: menuAddress("couch-primary"), WorkingPath: "/w/p", State: couchcore.ThreadParked}
	state := openRebootConfirmation(t, row)
	live := row
	live.State = couchcore.ThreadLive
	state.Inventory = []couchcore.ActionableThreadSummary{live}
	state, _ = reduceConfirmationKey(state, PanelKey{Kind: KeyDown})
	next, effects := reduceConfirmationKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 || next.CurrentFrame().Kind == MenuFrameConfirmation {
		t.Fatalf("a reboot confirmation on a now-live row dispatched %+v / frame %+v", effects, next.CurrentFrame())
	}
	refreshed := reconcileMenuFrames(state, []couchcore.ActionableThreadSummary{row})
	if refreshed.CurrentFrame().Kind == MenuFrameConfirmation {
		t.Fatalf("a refresh kept a reboot confirmation the row no longer offers")
	}
}

// Reboot changes its own row: the :0 row leaves under its old tag and comes
// back under a new one, a slot row turns live. The frame whose own operation
// is running must survive that, or the operator reads "no longer applicable"
// over a reboot that went on to work.
func TestRebootFrameSurvivesItsOwnStateChange(t *testing.T) {
	confirmAndDispatch := func(t *testing.T, row couchcore.ActionableThreadSummary) MenuState {
		state := openRebootConfirmation(t, row)
		state, _ = reduceConfirmationKey(state, PanelKey{Kind: KeyDown})
		state, effects := reduceConfirmationKey(state, PanelKey{Kind: KeyEnter})
		if len(effects) != 1 || effects[0].Operation != "reboot" || state.InFlight.Operation != "reboot" {
			t.Fatalf("reboot did not dispatch: %+v", effects)
		}
		return state
	}
	t.Run("primary vanishes and returns under a new tag", func(t *testing.T) {
		row := couchcore.ActionableThreadSummary{Address: menuAddress("couch-old"), WorkingPath: "/w/p", State: couchcore.ThreadParked}
		state := confirmAndDispatch(t, row)
		fresh := couchcore.ActionableThreadSummary{Address: menuAddress("couch-new"), WorkingPath: "/w/p", State: couchcore.ThreadLive}
		state, previous := replaceMenuInventory(state, []couchcore.ActionableThreadSummary{fresh})
		state = reconcileMenuFrames(state, previous)
		if frame := state.CurrentFrame(); frame.Kind != MenuFrameConfirmation || frame.Action != "reboot" || state.Notice.Level == MenuNoticeError || strings.Contains(state.Notice.Text, "no longer") {
			t.Fatalf("frame %v/%q notice %q", frame.Kind, frame.Action, state.Notice.Text)
		}
		// Its result lands under the NEW tag and still completes the attempt.
		next := reduceOperationResult(state, MenuEvent{Operation: "reboot", Attempt: state.InFlight.Attempt, Success: true, Address: fresh.Address})
		if next.InFlight.Operation != "" || next.CurrentFrame().Kind != MenuFrameRoot || next.Frames[0].SelectedAddress != fresh.Address {
			t.Fatalf("result not consumed: in flight %+v, frame %v, selected %+v", next.InFlight, next.CurrentFrame().Kind, next.Frames[0].SelectedAddress)
		}
	})
	t.Run("slot row turns live", func(t *testing.T) {
		row := menuSlotRow(1, "couch-old")
		row.State, row.Reason = couchcore.ThreadParked, ""
		state := confirmAndDispatch(t, row)
		live := menuSlotRow(1, "couch-new")
		live.State, live.Reason = couchcore.ThreadLive, ""
		state, previous := replaceMenuInventory(state, []couchcore.ActionableThreadSummary{live})
		state = reconcileMenuFrames(state, previous)
		if frame := state.CurrentFrame(); frame.Kind != MenuFrameConfirmation || frame.Action != "reboot" || strings.Contains(state.Notice.Text, "no longer") {
			t.Fatalf("frame %v/%q notice %q", frame.Kind, frame.Action, state.Notice.Text)
		}
	})
}

// The switcher filter matches what a row shows -- its label and the agent's
// published summary -- and never a stored name (pair#363).
func TestSwitcherFilterMatchesLabelAndPublishedSummary(t *testing.T) {
	primary := couchcore.ActionableThreadSummary{Address: menuAddress("couch-primary"), WorkingPath: "/w/repo", Name: "renamed", Description: "described", PublishedSummary: "fixing the parser", State: couchcore.ThreadLive}
	slot := menuSlotRow(1, "couch-slot")
	slot.State, slot.Reason, slot.Name = couchcore.ThreadLive, "", "slotname"
	state := NewMenuState([]couchcore.ActionableThreadSummary{primary, slot}, primary.Address)
	for filter, want := range map[string]couchcore.ThreadRowKey{
		"parser": menuRowKey(primary),
		"repo":   menuRowKey(primary),
		"pair:1": menuRowKey(slot),
	} {
		state.Frames[0].Filter = filter
		got := VisibleMenuThreads(state)
		if len(got) != 1 || menuRowKey(got[0]) != want {
			t.Errorf("filter %q selected %+v", filter, got)
		}
	}
	for _, filter := range []string{"renamed", "described", "slotname"} {
		state.Frames[0].Filter = filter
		if got := VisibleMenuThreads(state); len(got) != 0 {
			t.Errorf("filter %q matched a stored field: %+v", filter, got)
		}
	}
}

// A path-missing row says what the reboot result says, per kind, in its
// status and on Enter. A :1+ (which offers nothing) says add slot recreates
// its directory (couchcore.RebootDirectoryMissing); a :0 offers reboot, which
// archives the record, and its checkout is what brings it back
// (couchcore.RebootCheckoutMissing) -- add slot makes :1+ slots and a :0 row
// never offers it here (pair#363 M2 review).
func TestPathMissingRowExplainsItsNextStep(t *testing.T) {
	primary := couchcore.ActionableThreadSummary{Address: menuAddress("couch-primary"), WorkingPath: "/w/gone", State: couchcore.ThreadUnusable, Reason: couchcore.ReasonPathMissing}
	slot := menuSlotRow(1, "couch-slot")
	slot.Reason = couchcore.ReasonPathMissing
	for _, tc := range []struct {
		row  couchcore.ActionableThreadSummary
		want string
	}{
		{primary, "checkout missing — reboot archives this record; restore the checkout to start here again"},
		{slot, "directory missing — add slot recreates it"},
	} {
		row := tc.row
		if got := rootStateText(row, time.Now()); !strings.Contains(got, tc.want) {
			t.Errorf("%s status = %q, want %q", row.Label(), got, tc.want)
		}
		state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
		state.InventoryReady = true
		next, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
		if len(effects) != 0 || !strings.Contains(next.Notice.Text, tc.want) {
			t.Errorf("%s Enter: effects %v, notice %q", row.Label(), effects, next.Notice.Text)
		}
	}
}

// An unknown row has no verdict this round and offers nothing (resolved
// ambiguity 5); its status says so rather than reading like progress.
func TestUnknownRowSaysItsStateCouldNotBeChecked(t *testing.T) {
	row := couchcore.ActionableThreadSummary{Address: menuAddress("couch-primary"), WorkingPath: "/w/p", State: couchcore.ThreadUnusable, Reason: couchcore.ReasonUnknown}
	if got := rootStateText(row, time.Now()); got != "state could not be checked" {
		t.Fatalf("unknown status = %q", got)
	}
}

// The rule behind three findings in one family (lessons:
// refusal-names-unoffered-action): every row-facing next step names only an
// action that row's kind can reach. Checked over the derived row domain, for
// the advice authority (menuRowAdviceOf) and for the surfaces that print it --
// the status column, Enter's refusal and every confirmation the row offers --
// so hand-written advice that bypasses the authority is caught too.
//
// "Names" is a whole-word match on every action any row offers, by id and by
// display label, so the vocabulary grows with the table. An action reached on
// another row (OnPrimary: add slot on the live :0 recreating a gone :1+) is
// checked against that row's offer instead, and only a :1+ may defer to it.
func TestRowAdviceNamesOnlyReachableActions(t *testing.T) {
	shapes := everyMenuRowShape(t)
	vocabulary := map[string]bool{}
	var livePrimary []string
	for _, s := range shapes {
		offered := menuActionItems(s.row)
		for _, action := range offered {
			vocabulary[action] = true
		}
		if !s.slot && s.state == couchcore.ThreadLive && s.phase == "" {
			livePrimary = offered
		}
	}
	if !vocabulary["add-slot"] || !vocabulary["reboot"] || !vocabulary["resume"] || len(livePrimary) == 0 {
		t.Fatalf("derived vocabulary %v / live :0 offer %v is missing the actions this sweep exists to check", vocabulary, livePrimary)
	}
	names := func(text, action string) bool {
		for _, word := range []string{action, menuItemLabel(action)} {
			if regexp.MustCompile(`(^|[^\pL\pN-])` + regexp.QuoteMeta(word) + `($|[^\pL\pN-])`).MatchString(text) {
				return true
			}
		}
		return false
	}
	check := func(shape, where string, step menuNextStep, slot bool, offered []string) {
		reach := offered
		if step.OnPrimary {
			if !slot {
				t.Errorf("%s %s: a :0 row defers %q to the primary, which is itself", shape, where, step.Text)
			}
			reach = livePrimary
		}
		for action := range vocabulary {
			if names(step.Text, action) && !slices.Contains(reach, action) {
				t.Errorf("%s %s: %q names %s, which is not offered there (offered %v)", shape, where, step.Text, action, reach)
			}
		}
	}
	for _, s := range shapes {
		f := menuRowFactsOf(s.row)
		offered := menuRowActions(f)
		advice := menuRowAdviceOf(f)
		check(s.name, "notice", advice.Notice, s.slot, offered)
		check(s.name, "enter", advice.Enter, s.slot, offered)
		check(s.name, "reboot cost", advice.RebootCost, s.slot, offered)
		// A reboot suffix only shows in reboot's confirmation; on a row that
		// offers no reboot it is text nobody can see (#363 M3 review).
		if advice.RebootCost.Text != "" && !slices.Contains(offered, "reboot") {
			t.Errorf("%s: reboot cost %q on a row that offers no reboot (offered %v)", s.name, advice.RebootCost.Text, offered)
		}

		// The surfaces print the authority's words, and nothing else of
		// their own that names an action.
		if advice.Notice.Text != "" && !strings.Contains(rootStateText(s.row, time.Now()), advice.Notice.Text) {
			t.Errorf("%s: status %q does not carry the advice %q", s.name, rootStateText(s.row, time.Now()), advice.Notice.Text)
		}
		if !s.row.Live() && enterOperationFor(s.row) == "" {
			refusal := enterRefusalNotice(s.row)
			if !strings.Contains(refusal, advice.Enter.Text) {
				t.Errorf("%s: Enter refusal %q does not carry the advice %q", s.name, refusal, advice.Enter.Text)
			}
			// The explanation half is the notice, already checked; the rest
			// is the way forward.
			check(s.name, "enter refusal", menuNextStep{Text: strings.TrimPrefix(refusal, s.row.Label()+": "+unusableThreadNotice(s.row))}, s.slot, offered)
		}
		state := NewMenuState([]couchcore.ActionableThreadSummary{s.row}, s.row.Address)
		for _, action := range offered {
			frame := MenuFrame{Kind: MenuFrameConfirmation, RowKey: menuRowKey(s.row), Thread: s.row.Address, Action: action}
			items := confirmationMenuItems(state, frame)
			item := strings.TrimPrefix(items[len(items)-1], action+" "+s.row.Label())
			check(s.name, action+" confirmation", menuNextStep{Text: item}, s.slot, offered)
		}
	}
}

// The switcher names the orphaned server's pid, in the same sentence every
// other surface prints (#399).
func TestUnusableNoticeNamesTheOrphanedServer(t *testing.T) {
	row := couchcore.ActionableThreadSummary{Address: menuAddress("couch-orphan"), WorkingPath: "/w/p",
		State: couchcore.ThreadUnusable, Reason: couchcore.ReasonOrphanedServer,
		Orphan: &launcher.SessionServerIdentity{PID: 812, Session: "📁1-37"}}
	if got, want := unusableThreadNotice(row), launcher.OrphanDiagnostic("📁1-37", 812); got != want {
		t.Fatalf("notice = %q, want %q", got, want)
	}
}

// The reap confirmation names the exact server it ends and what goes with it:
// the agent under it may still be writing (#399).
func TestReapConfirmationNamesTheServerAndItsTree(t *testing.T) {
	row := couchcore.ActionableThreadSummary{Address: menuAddress("couch-orphan"), WorkingPath: "/w/p",
		State: couchcore.ThreadUnusable, Reason: couchcore.ReasonOrphanedServer,
		Orphan: &launcher.SessionServerIdentity{PID: 812, Session: "📁1-37"}}
	text := reapConfirmationCost(row)
	for _, want := range []string{"PID 812", "everything under it", "the agent"} {
		if !strings.Contains(text, want) {
			t.Fatalf("confirmation %q lacks %q", text, want)
		}
	}
}

// #399 M2 review BR-9: a successful reap closes its confirmation and completes
// the attempt; before, the reducer had no arm for it and the frame lingered.
func TestReapSuccessClosesItsConfirmation(t *testing.T) {
	row := couchcore.ActionableThreadSummary{Address: menuAddress("couch-orphan"), WorkingPath: "/w/p",
		State: couchcore.ThreadUnusable, Reason: couchcore.ReasonOrphanedServer,
		Orphan: &launcher.SessionServerIdentity{PID: 812, Session: "📁1-37"}}
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state.InventoryReady = true
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state.Frames[len(state.Frames)-1].SelectedItem = "reap"
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	if state.CurrentFrame().Kind != MenuFrameConfirmation || state.CurrentFrame().Action != "reap" {
		t.Fatalf("reap did not confirm: %+v", state.CurrentFrame())
	}
	state, _ = reduceConfirmationKey(state, PanelKey{Kind: KeyDown})
	state, effects := reduceConfirmationKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "reap" {
		t.Fatalf("reap did not dispatch: %+v", effects)
	}
	next := reduceOperationResult(state, MenuEvent{Operation: "reap", Attempt: state.InFlight.Attempt, Success: true, Address: row.Address})
	if next.InFlight.Operation != "" || next.CurrentFrame().Kind != MenuFrameRoot || next.Notice.Level == MenuNoticeError {
		t.Fatalf("reap result not consumed: in flight %+v, frame %v, notice %q", next.InFlight, next.CurrentFrame().Kind, next.Notice.Text)
	}
}
