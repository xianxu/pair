package couchtty

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func aliasStart576(state MenuState, path string, target couchcore.ThreadTarget) (MenuState, []MenuEffect) {
	state.Frames = append([]MenuFrame(nil), state.Frames...)
	state.Frames[len(state.Frames)-1].PreviewResolution = couchcore.StartResolution{Worktree: couchcore.Worktree(path), Target: target}
	return dispatchMenuOperation(state, MenuEffect{Operation: "start"}, couchcore.ThreadAddress{})
}

func requireAliasBusy576(t *testing.T, state MenuState, effects []MenuEffect, owner string) {
	t.Helper()
	if len(effects) != 0 || !strings.Contains(state.Notice.Text, "busy: "+owner) {
		t.Fatalf("want busy %s, effects=%+v notice=%+v", owner, effects, state.Notice)
	}
}

func TestMenuPrimaryStartAliasesOrdinaryRowsByResolvedScope(t *testing.T) {
	const path = "/alias-controls576/repo"
	first := groupedRow(path, 0, "first")
	second := groupedRow(path, 0, "second")
	scope, err := launcher.ResolveRepoScope(path)
	if err != nil {
		t.Fatal(err)
	}
	if first.Address.RepoScope != scope.Key {
		t.Fatal("fixture must use launcher's scope hash")
	}
	for _, operation := range []string{"reboot", "recover", "recovery-confirmation"} {
		t.Run(operation, func(t *testing.T) {
			for _, startFirst := range []bool{false, true} {
				t.Run(fmt.Sprint(startFirst), func(t *testing.T) {
					state := NewMenuState([]couchcore.ActionableThreadSummary{first, second}, first.Address)
					var effects []MenuEffect
					if startFirst {
						state, effects = aliasStart576(state, path, couchcore.ThreadTarget{})
						if len(effects) != 1 {
							t.Fatalf("start not admitted: %+v", state.Notice)
						}
						state, effects = dispatchThreadOperation(state, operation, first.Address)
						requireAliasBusy576(t, state, effects, "start")
					} else {
						state, effects = dispatchThreadOperation(state, operation, first.Address)
						if len(effects) != 1 {
							t.Fatalf("row operation not admitted: %+v", state.Notice)
						}
						state, effects = aliasStart576(state, path, couchcore.ThreadTarget{})
						requireAliasBusy576(t, state, effects, operation)
					}
				})
			}
		})
	}
}

func TestMenuStartScopeProjectsEveryMatchingOrdinaryRowAndBeatsOldResult(t *testing.T) {
	const path = "/alias-controls576/multiple"
	rows := []couchcore.ActionableThreadSummary{groupedRow(path, 0, "one"), groupedRow(path, 0, "two"), groupedRow(path, 0, "three")}
	state := NewMenuState(rows, rows[0].Address)
	state.OperationSequence = 99
	state.Operations = map[menuOperationTarget]MenuOperation{
		{RowKey: menuRowKey(rows[0])}: {Origin: MenuOperationOrigin{Attempt: 99, Operation: "relaunch", Address: rows[0].Address}, Phase: MenuOperationFailed, Detail: "old failure"},
	}
	state, effects := aliasStart576(state, path, couchcore.ThreadTarget{})
	if len(effects) != 1 {
		t.Fatal("start not admitted")
	}
	projected := menuOperationRows(state)
	for _, row := range rows {
		job, ok := menuOperationForRow(state, row)
		if !ok || job.Phase != MenuOperationRunning || job.Origin.Attempt != effects[0].Attempt {
			t.Fatalf("row %s did not select running alias: %+v", row.Address.Tag, job)
		}
		rendered, ok := projected[menuRowKey(row)]
		if !ok || rendered.Phase != MenuOperationRunning || rendered.Origin.Attempt != effects[0].Attempt {
			t.Fatalf("row %s missing running projection: %+v", row.Address.Tag, rendered)
		}
		selected, busy := selectBusyMenuRow(state, row)
		if !busy || selected.LandingAttempt != effects[0].Attempt {
			t.Fatalf("row %s not selectable as shared owner", row.Address.Tag)
		}
		blocked, effects := dispatchThreadOperation(state, "relaunch", row.Address)
		requireAliasBusy576(t, blocked, effects, "start")
	}
}

func TestMenuNumberedStartUsesExactCheckoutAlias(t *testing.T) {
	row := groupedRow("/alias-controls576/numbered", 1, "one")
	replacement := row
	replacement.Address.Tag = "replacement"
	for _, startFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(startFirst), func(t *testing.T) {
			state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
			var effects []MenuEffect
			if startFirst {
				state, effects = aliasStart576(state, row.WorkingPath, row.Target)
				if len(effects) != 1 {
					t.Fatal("numbered start not admitted")
				}
				state.Inventory = []couchcore.ActionableThreadSummary{replacement}
				state, effects = dispatchMenuOperation(state, MenuEffect{Operation: "relaunch", RowKey: row.RowKey}, replacement.Address)
				requireAliasBusy576(t, state, effects, "start")
			} else {
				state, effects = dispatchMenuOperation(state, MenuEffect{Operation: "relaunch", RowKey: row.RowKey}, row.Address)
				if len(effects) != 1 {
					t.Fatal("numbered row operation not admitted")
				}
				state, effects = aliasStart576(state, row.WorkingPath, row.Target)
				requireAliasBusy576(t, state, effects, "relaunch")
			}
		})
	}
}

func TestMenuStartAliasNegativeIdentityControls(t *testing.T) {
	const path = "/alias-controls576/negative"
	sameCWD := groupedRow(path, 0, "wrong-scope")
	sameCWD.Address.RepoScope = "different-scope"
	sameCWD.RowKey = couchcore.ThreadRowKey{Kind: couchcore.ThreadTargetOrdinary, Address: sameCWD.Address}
	sameCWD.Target = couchcore.ThreadTarget{Kind: couchcore.ThreadTargetOrdinary, Address: sameCWD.Address}
	prefix := groupedRow(path+"-other", 0, "prefix")
	numbered := groupedRow(path, 1, "numbered")
	// Even a reused scope is not authority to conflate a primary and a slot row.
	scope, _ := launcher.ResolveRepoScope(path)
	numbered.Address.RepoScope = scope.Key
	controls := []struct {
		name      string
		row       couchcore.ActionableThreadSummary
		startPath string
		target    couchcore.ThreadTarget
	}{
		{"matching-cwd-different-scope", sameCWD, path, couchcore.ThreadTarget{}},
		{"path-prefix", prefix, path, couchcore.ThreadTarget{}},
		{"primary-scope-versus-numbered", numbered, path, couchcore.ThreadTarget{}},
		{"different-numbered-checkout", groupedRow(path, 2, "two"), numbered.WorkingPath, numbered.Target},
	}
	for _, tc := range controls {
		t.Run(tc.name, func(t *testing.T) {
			state := NewMenuState([]couchcore.ActionableThreadSummary{tc.row}, tc.row.Address)
			state, effects := aliasStart576(state, tc.startPath, tc.target)
			if len(effects) != 1 {
				t.Fatal("start not admitted")
			}
			if job, ok := menuOperationForRow(state, tc.row); ok && job.Phase == MenuOperationRunning {
				t.Fatalf("negative identity matched start: %+v", job)
			}
			if job, ok := menuOperationRows(state)[menuRowKey(tc.row)]; ok && job.Phase == MenuOperationRunning {
				t.Fatalf("negative identity projected busy: %+v", job)
			}
			_, effects = dispatchMenuOperation(state, MenuEffect{Operation: "relaunch", RowKey: menuRowKey(tc.row)}, tc.row.Address)
			if len(effects) != 1 {
				t.Fatal("independent target incorrectly refused")
			}
		})
	}
}

func TestMenuOrdinaryActionsKeepExactAddressIndependence(t *testing.T) {
	first := groupedRow("/alias-controls576/independent", 0, "one")
	second := groupedRow("/alias-controls576/independent", 0, "two")
	for _, operation := range []string{"relaunch", "resume", "detach", "reload-context", "switch-agent"} {
		t.Run(operation, func(t *testing.T) {
			state := NewMenuState([]couchcore.ActionableThreadSummary{first, second}, first.Address)
			state, effects := dispatchThreadOperation(state, operation, first.Address)
			if len(effects) != 1 {
				t.Fatal("first not admitted")
			}
			_, effects = dispatchThreadOperation(state, operation, second.Address)
			if len(effects) != 1 {
				t.Fatal("ordinary independent address blocked")
			}
		})
	}
}

func TestMenuStartAliasesMaterializeLinearlyWithoutRekeyingRunningOwners(t *testing.T) {
	for _, n := range []int{32, 64} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			rows := make([]couchcore.ActionableThreadSummary, n)
			originals := make(map[menuOperationTarget]MenuOperation, n)
			for i := range rows {
				path := fmt.Sprintf("/alias-controls576/cost/%d", i)
				rows[i] = groupedRow(path, 0, fmt.Sprint(i))
				originals[menuOperationTarget{StartPath: path}] = MenuOperation{Origin: MenuOperationOrigin{Operation: "start", Attempt: uint64(i + 1)}, Phase: MenuOperationRunning}
			}
			state := NewMenuState(rows, rows[0].Address)
			state.Operations = originals
			var projectWork, pruneWork menuOperationWork
			projected := menuOperationRowsMeasured(state, &projectWork)
			pruned := pruneMenuOperationsMeasured(state, &pruneWork)
			want := menuOperationWork{Rows: n, Jobs: n}
			if projectWork != want || pruneWork != want {
				t.Fatalf("N=%d projection=%+v prune=%+v want=%+v", n, projectWork, pruneWork, want)
			}
			if len(projected) != n {
				t.Fatalf("projected%d rows want%d", len(projected), n)
			}
			for _, row := range rows {
				if job, ok := projected[menuRowKey(row)]; !ok || job.Phase != MenuOperationRunning {
					t.Fatalf("scope alias missing for %+v", row.Address)
				}
			}
			if !reflect.DeepEqual(pruned.Operations, originals) || !reflect.DeepEqual(state.Operations, originals) {
				t.Fatal("materialization/pruning rekeyed or mutated active ownership")
			}
		})
	}
}

func TestMenuScopeWideReservationConflictsSymmetricallyWithExactOrdinaryJob(t *testing.T) {
	const path = "/alias-controls576/symmetric"
	first := groupedRow(path, 0, "one")
	second := groupedRow(path, 0, "two")
	for _, wide := range []string{"start", "reboot", "recover", "recovery-confirmation"} {
		t.Run(wide, func(t *testing.T) {
			for _, wideFirst := range []bool{false, true} {
				t.Run(fmt.Sprint(wideFirst), func(t *testing.T) {
					state := NewMenuState([]couchcore.ActionableThreadSummary{first, second}, first.Address)
					var effects []MenuEffect
					admitWide := func(s MenuState) (MenuState, []MenuEffect) {
						if wide == "start" {
							return aliasStart576(s, path, couchcore.ThreadTarget{})
						}
						return dispatchThreadOperation(s, wide, second.Address)
					}
					if wideFirst {
						state, effects = admitWide(state)
						if len(effects) != 1 {
							t.Fatal("scope owner not admitted")
						}
						state, effects = dispatchThreadOperation(state, "relaunch", first.Address)
						requireAliasBusy576(t, state, effects, wide)
					} else {
						state, effects = dispatchThreadOperation(state, "relaunch", first.Address)
						if len(effects) != 1 {
							t.Fatal("exact owner not admitted")
						}
						state, effects = admitWide(state)
						requireAliasBusy576(t, state, effects, "relaunch")
					}
				})
			}
		})
	}
}
