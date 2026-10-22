package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rivo/uniseg"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testcouch"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

const drainAcceptanceGraceful = "Drain: finish your current atomic step and wait for background work that is already running to finish. Leave queued, not-started work queued and note it in the issue Log as outstanding." + drainAcceptanceGuidance
const drainAcceptanceAck = "acknowledge message received, will resume processing after drain is lifted"
const drainAcceptanceLift = "drain lifted; continue your current work\nIf you received no drain notice, no checkpoint or recovery action is required."
const drainAcceptanceGuidance = " Write where you stopped and the next step in the issue Log; commit and push the checkpoint, then end your turn and stay idle. Start no new work. While idle, keep receiving messages. For new work, reply once: 'acknowledge message received, will resume processing after drain is lifted'. Do not create courtesy acknowledgements or acknowledgement loops."

func TestDrainAcceptanceGracefulBody(t *testing.T) {
	if got := couchcore.DrainNotice(false); got != drainAcceptanceGraceful {
		t.Fatalf("production notice differs from independent native expected body:\ngot:  %s\nwant: %s", got, drainAcceptanceGraceful)
	}
}

// TestDisposableCouchDrainFlags qualifies drain policy through the landed native
// fleet owner. Its actors, native topology, inherited CLI environment and exact
// generation-qualified teardown belong to testcouch, not this scenario.
func TestDisposableCouchDrainFlags(t *testing.T) {
	if os.Getenv("PAIR_DRAIN_ACCEPTANCE") != "1" {
		t.Skip("opt in with PAIR_DRAIN_ACCEPTANCE=1 or make test-drain-acceptance")
	}
	binaries := testcouch.Build(t)
	for _, personality := range []string{"claude", "codex"} {
		if !t.Run(personality, func(t *testing.T) {
			fleet := testcouch.New(t, binaries, testcouch.Options{Workers: 2, AgentKind: personality, RuntimeBudget: 90 * time.Second})
			// Register after New so this assertion runs after the scenario's cleanup,
			// while New's own fallback cleanup remains available on every failure.
			t.Cleanup(func() {
				cleanup, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				if err := fleet.Close(cleanup); err != nil {
					t.Errorf("owned fleet cleanup: %v", err)
					return
				}
				if err := fleet.AssertCleaned(); err != nil {
					t.Error(err)
				}
			})
			setup, cancelSetup := context.WithTimeout(t.Context(), 180*time.Second)
			err := fleet.Prepare(setup)
			cancelSetup()
			if err != nil {
				t.Fatal(err)
			}
			legacyPath := filepath.Join(fleet.Roots.Mutable["store"], "drain.json")
			legacy := []byte("opaque legacy operator evidence; never interpret or rewrite\n")
			if err := os.WriteFile(legacyPath, legacy, 0600); err != nil {
				t.Fatal(err)
			}
			verifyDisk := func() {
				t.Helper()
				raw, err := os.ReadFile(legacyPath)
				if err != nil || !bytes.Equal(raw, legacy) {
					t.Fatalf("legacy drain bytes changed: %v %q", err, raw)
				}
				err = filepath.WalkDir(fleet.Roots.Mutable["store"], func(path string, entry os.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if !entry.IsDir() && strings.Contains(strings.ToLower(entry.Name()), "drain") && path != legacyPath {
						return fmt.Errorf("new drain persistence: %s", path)
					}
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			started := time.Now()
			ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
			defer cancel()
			if err := fleet.Start(ctx); err != nil {
				t.Fatal(err)
			}
			for n := 1; n <= 2; n++ {
				if err := fleet.StartWorker(ctx, n); err != nil {
					t.Fatal(err)
				}
			}
			actors, err := fleet.Actors(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(actors.Actors) != 3 {
				t.Fatalf("genuine cohort: %+v", actors)
			}
			seen := map[string]bool{}
			for _, actor := range actors.Actors {
				seen[actor.Binding.Slot] = true
			}
			for n := 0; n <= 2; n++ {
				if !seen[fmt.Sprintf("fixture:%d", n)] {
					t.Fatalf("missing genuine actor%d: %+v", n, actors)
				}
				ready, err := fleet.Agent(n)
				if err != nil || ready.Agent != personality {
					t.Fatalf("actor%d personality: %+v %v", n, ready, err)
				}
			}
			t.Logf("drain native manifest: roots=%+v run=%s source=%s binaries=%+v runtime_budget=90s", fleet.Roots, fleet.RunID, binaries.Source, binaries.Evidence)
			call := func(actor int, args ...string) string {
				t.Helper()
				result, err := fleet.ActorCLI(ctx, actor, args...)
				if err != nil || result.ExitCode != 0 {
					t.Fatalf("actor%d CLI%v: %+v %v; screen=%s", actor, args, result, err, fleet.Screen())
				}
				t.Logf("drain public CLI actor=%d argv=%q stdout=%s stderr=%s", actor, args, result.Stdout, result.Stderr)
				return result.Stdout
			}
			response := func(actor int, args ...string) couchmessage.Response {
				t.Helper()
				raw := call(actor, args...)
				var result couchmessage.Response
				if err := json.Unmarshal([]byte(raw), &result); err != nil {
					t.Fatalf("public response: %v %s", err, raw)
				}
				if result.Error != "" {
					t.Fatalf("public refusal: %+v", result)
				}
				return result
			}
			flags := func(result couchmessage.Response, count int) {
				t.Helper()
				if err := drainAcceptanceFlags(result, count); err != nil {
					t.Fatal(err)
				}
				t.Logf("drain observed flag rows (requested posture only, never readiness): %+v", result.Drain.Rows)
			}
			accepted := func(actor int, id, body string) {
				t.Helper()
				var last couchmessage.Response
				err := waitFleet(ctx, func() (bool, error) {
					result, e := fleet.ActorCLI(ctx, 0, "--message-status", id, "--json")
					if e != nil {
						return false, e
					}
					if result.ExitCode != 0 {
						return false, fmt.Errorf("receipt status exit%d: %s %s", result.ExitCode, result.Stdout, result.Stderr)
					}
					if e := json.Unmarshal([]byte(result.Stdout), &last); e != nil {
						return false, e
					}
					if last.Error != "" || last.Receipt == nil {
						return false, fmt.Errorf("receipt missing/refused: %+v", last)
					}
					if last.Receipt.Message.Body != body {
						return false, fmt.Errorf("receipt body=%q want=%q", last.Receipt.Message.Body, body)
					}
					if last.Receipt.Status.Terminal() && last.Receipt.Status != couchmessage.Submitted {
						return false, fmt.Errorf("delivery terminal refusal: %+v", last.Receipt)
					}
					return last.Receipt.Status == couchmessage.Submitted, nil
				})
				if err != nil {
					t.Fatalf("receipt%s: %v", id, err)
				}
				t.Logf("drain submitted receipt: %+v", last.Receipt)
				err = waitFleet(ctx, func() (bool, error) {
					inputs, err := fleet.Accepted(actor)
					if errors.Is(err, os.ErrNotExist) {
						return false, nil
					}
					if err != nil {
						return false, err
					}
					for _, input := range inputs {
						if strings.Contains(input.Body, "delivery "+id+"]\n"+body) {
							t.Logf("drain independently accepted: actor=%d receipt=%s input=%+v", actor, id, input)
							return true, nil
						}
					}
					return false, nil
				})
				if err != nil {
					t.Fatalf("accepted body actor%d receipt%s: %v", actor, id, err)
				}
			}
			notices := func(result couchmessage.Response, body string) {
				t.Helper()
				if result.Operation == nil || result.Operation.Drain == nil {
					t.Fatalf("notice operation missing: %+v", result)
				}
				id := result.Operation.ID
				err := waitFleet(ctx, func() (bool, error) {
					result = response(0, "--operation-status", id, "--json")
					if result.Operation == nil || result.Operation.Drain == nil {
						return false, fmt.Errorf("notice operation disappeared: %+v", result)
					}
					if result.Operation.Status == couchmessage.ReceiptFailed {
						return false, fmt.Errorf("notice operation failed: %+v", result.Operation)
					}
					return result.Operation.Status == couchmessage.ReceiptSucceeded, nil
				})
				if err != nil {
					t.Fatal(err)
				}
				if len(result.Operation.Drain.Children) != 2 {
					t.Fatalf("two worker notices required: %+v", result)
				}
				recipients := map[string]bool{}
				for _, delivery := range result.Operation.Drain.Children {
					if delivery.Phase != couchmessage.DrainChildDelivered || delivery.ID == "" {
						t.Fatalf("notice admission: %+v", delivery)
					}
					n := 0
					switch delivery.Workspace {
					case "fixture:1":
						n = 1
					case "fixture:2":
						n = 2
					default:
						t.Fatalf("unexpected recipient: %+v", delivery)
					}
					if recipients[delivery.Workspace] {
						t.Fatalf("duplicate notice recipient: %+v", result)
					}
					recipients[delivery.Workspace] = true
					accepted(n, delivery.ID, body)
				}
			}
			send := func(from, to int, body string) {
				t.Helper()
				raw := call(from, "--send-to", fmt.Sprintf("fixture:%d", to), "--message", body)
				fields := strings.Fields(raw)
				if len(fields) == 0 {
					t.Fatalf("send has no public receipt: %q", raw)
				}
				accepted(to, fields[0], body)
			}
			flags(response(0, "--drain", "--status", "--json"), 0)
			graceful := drainAcceptanceGraceful
			fast := "Drain (fast): stop now; do not finish the current step. Promptly stop your own background work via supported controls; preserve unfinished working-tree edits in your checkpoint." + drainAcceptanceGuidance
			result := response(0, "--drain", "--json")
			flags(response(0, "--drain", "--status", "--json"), 2)
			notices(result, graceful)
			// Read-only status also works from a flagged, genuinely registered worker.
			flags(response(1, "--drain", "--status", "--json"), 2)
			send(0, 1, "ordinary-while-flagged-"+fleet.RunID)
			send(1, 0, drainAcceptanceAck)
			// Native Pair splits this width between panes; keep the agent fixture
			// above its 20-column minimum. Pure renderer tests cover widths 1–139.
			var frame string
			for _, cols := range []uint16{80, 120} {
				if err := fleet.Resize(32, cols); err != nil {
					t.Fatal(err)
				}
				frame, err = waitDrainAcceptanceScreen(ctx, fleet.Screen, func(screen string) bool {
					_, err := drainAcceptanceBadge(screen, int(cols))
					return err == nil
				})
				if err != nil {
					t.Fatalf("current badge at width%d: %v; screen=%s", cols, err, fleet.Screen())
				}
				t.Logf("drain current frame width%d:\n%s", cols, frame)
			}
			// Click the ordinary worker-1 chip in current cell coordinates. The shared
			// renderer compresses subsequent same-repo chips to :N; no byte offsets.
			row, err := drainAcceptanceBadge(frame, 120)
			if err != nil {
				t.Fatal(err)
			}
			at := strings.Index(row, ":1")
			if at < 0 {
				t.Fatalf("worker1 chip absent from current row: %q", row)
			}
			col := uniseg.GraphemeClusterCount(row[:at]) + 1
			if err := fleet.Input([]byte(fmt.Sprintf("\x1b[<0;%d;32M\x1b[<0;%d;32m", col, col))); err != nil {
				t.Fatal(err)
			}
			frame, err = waitDrainAcceptanceScreen(ctx, fleet.Screen, func(screen string) bool {
				row, err := drainAcceptanceBadge(screen, 120)
				return err == nil && (strings.Contains(row, "[:1") || strings.Contains(row, "[fixture:1"))
			})
			if err != nil {
				t.Fatalf("ordinary flagged worker selection: %v; screen=%s", err, fleet.Screen())
			}
			t.Logf("drain ordinary selected worker frame:\n%s", frame)
			result = response(0, "--undrain", "--json")
			flags(response(0, "--drain", "--status", "--json"), 0)
			notices(result, drainAcceptanceLift)
			result = response(0, "--drain", "--fast", "--json")
			flags(response(0, "--drain", "--status", "--json"), 2)
			notices(result, fast)
			send(0, 2, "ordinary-while-fast-flagged-"+fleet.RunID)
			send(2, 0, drainAcceptanceAck)
			result = response(0, "--undrain", "--message", "custom-continuation-marker", "--json")
			flags(response(0, "--drain", "--status", "--json"), 0)
			notices(result, drainAcceptanceLift+"\nNote: "+strconv.Quote("custom-continuation-marker"))
			result = response(0, "--drain", "--json")
			flags(response(0, "--drain", "--status", "--json"), 2)
			notices(result, graceful)
			// Use disambiguated CSI-u physical keys. Ctrl+Space first changes focus
			// to Couch's switcher; Alt+x then requests its normal leave confirmation.
			if err := fleet.Input([]byte("\x1b[32;5u")); err != nil {
				t.Fatal(err)
			}
			if err := waitFleet(ctx, func() (bool, error) {
				return strings.Contains(fleet.Screen(), "Ctrl+Y copy last notice / selected error"), nil
			}); err != nil {
				t.Fatal(err)
			}
			if err := fleet.Input([]byte("\x1b[120;3u")); err != nil {
				t.Fatal(err)
			}
			frame, err = waitDrainAcceptanceScreen(ctx, fleet.Screen, func(screen string) bool {
				return strings.Contains(screen, "threads › leave couch") && strings.Contains(screen, "cancel") && strings.Contains(screen, "parking")
			})
			if err != nil {
				t.Fatalf("normal quit confirmation while flagged: %v; screen=%s", err, fleet.Screen())
			}
			t.Logf("drain normal quit confirmation (never confirmed):\n%s", frame)
			if err := fleet.Input([]byte("\x1b[27u")); err != nil {
				t.Fatal(err)
			}
			if err := waitFleet(ctx, func() (bool, error) { return !strings.Contains(fleet.Screen(), "threads › leave couch"), nil }); err != nil {
				t.Fatal(err)
			}
			flags(response(0, "--drain", "--status", "--json"), 2)
			verifyDisk()
			// Close the switcher before the shared Detach helper opens it itself.
			if err := fleet.Input([]byte("\x1b[27u")); err != nil {
				t.Fatal(err)
			}
			if err := waitFleet(ctx, func() (bool, error) {
				return !strings.Contains(fleet.Screen(), "Ctrl+Y copy last notice / selected error"), nil
			}); err != nil {
				t.Fatal(err)
			}
			if err := fleet.Detach(ctx); err != nil {
				t.Fatal(err)
			}
			if err := fleet.RestartSupervisor(ctx); err != nil {
				t.Fatal(err)
			}
			if err := fleet.WaitActors(ctx, 3); err != nil {
				t.Fatal(err)
			}
			flags(response(0, "--drain", "--status", "--json"), 0)
			send(0, 2, "ordinary-after-restart-"+fleet.RunID)
			verifyDisk()
			t.Logf("drain native completion: agent=%s run=%s elapsed=%s legacy_bytes_unchanged=true no_new_drain_file=true restart_flags=0", personality, fleet.RunID, time.Since(started))
		}) {
			break
		}
	}
}

// waitDrainAcceptanceScreen returns the screen that satisfied the predicate.
func waitDrainAcceptanceScreen(ctx context.Context, read func() string, accept func(string) bool) (string, error) {
	var matched string
	err := waitFleet(ctx, func() (bool, error) {
		screen := read()
		if !accept(screen) {
			return false, nil
		}
		matched = screen
		return true, nil
	})
	return matched, err
}

func TestDrainAcceptanceScreenKeepsValidatedSnapshot(t *testing.T) {
	const valid = "validated frame with badge and worker chip"
	reads := 0
	read := func() string {
		reads++
		if reads == 1 {
			return valid
		}
		return ""
	}
	got, err := waitDrainAcceptanceScreen(t.Context(), read, func(screen string) bool { return screen == valid })
	if err != nil || got != valid || reads != 1 {
		t.Fatalf("validated frame replaced by redraw: frame=%q reads=%d err=%v", got, reads, err)
	}
}

func drainAcceptanceFlags(result couchmessage.Response, count int) error {
	if result.Drain == nil || len(result.Drain.Rows) != count {
		return fmt.Errorf("flag rows=%+v, want%d", result.Drain, count)
	}
	seen := map[string]bool{}
	for _, row := range result.Drain.Rows {
		if row.Workspace != "fixture:1" && row.Workspace != "fixture:2" {
			return fmt.Errorf("unexpected flag recipient: %+v", row)
		}
		if seen[row.Workspace] {
			return fmt.Errorf("duplicate flag row: %+v", row)
		}
		seen[row.Workspace] = true
		// Unknown observations cannot acquire an idle duration from the flag alone.
		if row.IdleMillis != nil && (!row.Connected || row.Settled == nil || !*row.Settled || row.Picker == nil || *row.Picker || row.Background == nil || *row.Background || *row.IdleMillis < 0) {
			return fmt.Errorf("unsupported idle claim: %+v", row)
		}
	}
	return nil
}

func drainAcceptanceBadge(screen string, cols int) (string, error) {
	lines := strings.Split(strings.TrimSuffix(screen, "\n"), "\n")
	if len(lines) != 32 {
		return "", fmt.Errorf("current screen rows=%d, want32", len(lines))
	}
	row := lines[len(lines)-1]
	// Fleet.Screen serializes one grapheme per cell, including a literal
	// space for each wide-head continuation. Count those serialized cells,
	// rather than re-measuring wide heads and double-counting continuations.
	if cells := uniseg.GraphemeClusterCount(row); cells != cols {
		return row, fmt.Errorf("current screen cells=%d want%d", cells, cols)
	}
	if strings.Count(row, "drain") != 1 || !strings.Contains(row, "drain 2") || strings.ContainsAny(row, "{}") {
		return row, fmt.Errorf("expected one compact drain badge: %q", row)
	}
	return row, nil
}

// Negative controls keep scenario oracles fail-closed without running a fleet.
func TestDrainAcceptanceOracles(t *testing.T) {
	screen := func(row string, width int) string {
		return strings.Repeat(strings.Repeat(" ", width)+"\n", 31) + row + strings.Repeat(" ", width-textwidth.Width(row)) + "\n"
	}
	good := screen("drain 2  fixture:0 :1 [:2]", 120)
	// Each prefix projects ten cells, including any explicit continuation.
	for _, prefix := range []string{"drain 2 🔔 ", "drain 2 e\u0301 ", "drain 2 👩‍💻 "} {
		wide := strings.Repeat(strings.Repeat(" ", 120)+"\n", 31) + prefix + strings.Repeat(" ", 110) + "\n"
		if _, err := drainAcceptanceBadge(wide, 120); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := drainAcceptanceBadge(good, 120); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []struct {
		screen string
		width  int
	}{
		{good, 30},
		{screen("drain 2 drain 2", 120), 120},
		{screen("drain 2 fixture:1{draining}", 120), 120},
		{screen("fixture:0 :1 :2", 120), 120},
		{"drain 2\n", 120},
	} {
		if _, err := drainAcceptanceBadge(bad.screen, bad.width); err == nil {
			t.Fatalf("accepted invalid current badge: %q width%d", bad.screen, bad.width)
		}
	}
	unknown := couchmessage.Response{Drain: &couchcore.DrainBoard{Rows: []couchcore.DrainRow{{Workspace: "fixture:1", Connected: true}, {Workspace: "fixture:2", Connected: true}}}}
	if err := drainAcceptanceFlags(unknown, 2); err != nil {
		t.Fatal(err)
	}
	for _, rows := range [][]couchcore.DrainRow{
		{{Workspace: "fixture:0"}, {Workspace: "fixture:1"}},
		{{Workspace: "fixture:1"}, {Workspace: "fixture:1"}},
		{{Workspace: "fixture:1", IdleMillis: new(int64)}, {Workspace: "fixture:2"}},
	} {
		if err := drainAcceptanceFlags(couchmessage.Response{Drain: &couchcore.DrainBoard{Rows: rows}}, 2); err == nil {
			t.Fatalf("accepted invalid flag projection: %+v", rows)
		}
	}
	if err := drainAcceptanceFlags(couchmessage.Response{}, 0); err == nil {
		t.Fatal("missing status accepted as empty flags")
	}
}
