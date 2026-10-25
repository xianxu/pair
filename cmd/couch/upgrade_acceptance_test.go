package main

import (
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
	"github.com/xianxu/pair/cmd/internal/testcouch"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
	"github.com/xianxu/pair/cmd/internal/zellijquery"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
)

// TestDisposableCouchBuildUpgrade exercises real old writers and warm native
// sessions. An incompatible candidate fails this same test, without an xfail
// mode or any deletion of readiness evidence to manufacture a legacy session.
func TestDisposableCouchBuildUpgrade(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_COUCH_UPGRADE") != "1" {
		t.Skip("set PAIR_NATIVE_COUCH_UPGRADE=1 and explicit baseline/candidate source checkouts")
	}
	scratch := os.TempDir()
	if info, err := os.Stat("/Volumes/TestScratch"); err == nil && info.IsDir() {
		scratch = "/Volumes/TestScratch"
	}
	scratch, err := filepath.EvalSymlinks(scratch)
	if err != nil {
		t.Fatal(err)
	}
	baselineSource := os.Getenv("PAIR_COUCH_UPGRADE_BASE_SOURCE")
	candidateSource := os.Getenv("PAIR_COUCH_UPGRADE_CANDIDATE_SOURCE")
	baseline := upgradeBuild(t, baselineSource, scratch)
	candidate := baseline
	if candidateSource != baselineSource {
		candidate = upgradeBuild(t, candidateSource, scratch)
	}
	for _, kind := range []string{"claude", "codex"} {
		caseCleaned := false
		casePassed := t.Run(kind, func(t *testing.T) {
			fleet := testcouch.New(t, baseline, testcouch.Options{ExplicitHub: true, Workers: 1, AgentKind: kind, RuntimeBudget: 180 * time.Second, RetainStartupServerLog: true})
			cleaned := false
			defer func() {
				cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
				defer stop()
				if err := fleet.Close(cleanup); err != nil {
					t.Errorf("upgrade cleanup uncertain; root=%s: %v", fleet.Roots.Root, err)
					return
				}
				if err := fleet.AssertCleaned(); err != nil {
					t.Error(err)
					return
				}
				cleaned = true
				caseCleaned = true
				t.Log("upgrade cleanup: all exact generations absent, children and IO joined")
			}()
			// Stop the personality series after uncertain native cleanup.
			t.Cleanup(func() {
				if !cleaned {
					t.Fatalf("do not reuse uncertain upgrade root %s", fleet.Roots.Root)
				}
			})
			setup, stop := context.WithTimeout(t.Context(), 240*time.Second)
			err := fleet.Prepare(setup)
			stop()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
			defer cancel()
			if err := fleet.Start(ctx); err != nil {
				t.Fatalf("baseline startup: %v", err)
			}
			if err := fleet.StartWorker(ctx, 1); err != nil {
				t.Fatalf("baseline worker: %v", err)
			}
			before := snapshotUpgrade(t, ctx, fleet)
			t.Log("upgrade phase: baseline actors ready; proving physical selection and old drafts")
			for n := 0; n <= 1; n++ {
				slot := fmt.Sprintf("fixture:%d", n)
				upgradeSelect(t, ctx, fleet, before[slot].Binding.Tag)
				if err := fleet.Input([]byte("upgrade-" + slot)); err != nil {
					t.Fatal(err)
				}
				forceFleetWaitScreen(t, ctx, fleet, "upgrade-"+slot)
			}
			documents := upgradeDocuments(t, ctx, fleet)
			t.Log("upgrade phase: old documents registered; detaching old console")
			oldList := upgradeCLI(t, ctx, fleet, "--list")
			for _, actor := range before {
				if !strings.Contains(oldList, actor.Ready.CWD) {
					t.Fatalf("baseline list omitted %s: %s", actor.Binding.Slot, oldList)
				}
			}
			if err := fleet.Detach(ctx); err != nil {
				t.Fatal(err)
			}
			for _, actor := range before {
				upgradeClients(t, ctx, fleet, actor.Binding.Session, false)
			}
			t.Log("upgrade phase: old console detached and joined; replacing private runtime")
			if err := fleet.ReplaceBuild(ctx, candidate); err != nil {
				t.Fatal(err)
			}
			if err := fleet.RestartSupervisor(ctx); err != nil {
				t.Fatalf("candidate startup adoption: %v; screen=%s", err, fleet.Screen())
			}
			t.Log("upgrade phase: candidate broker responding; checking exact old cohort and native clients")
			after := snapshotUpgrade(t, ctx, fleet)
			if err := compareUpgradeCohort(before, after); err != nil {
				t.Fatalf("candidate adoption: %v", err)
			}
			// Bare startup attaches the configured hub; workers remain available
			// for the explicit resume/attach checks below.
			upgradeClients(t, ctx, fleet, after["fixture:0"].Binding.Session, true)
			list := upgradeCLI(t, ctx, fleet, "--list")
			for _, actor := range before {
				if !strings.Contains(list, actor.Ready.CWD) {
					t.Fatalf("candidate list omitted %s: %s", actor.Binding.Slot, list)
				}
				detail := upgradeCLI(t, ctx, fleet, "--show", actor.Binding.Slot)
				if !strings.Contains(detail, actor.Binding.Tag) || !strings.Contains(detail, actor.Binding.Scope) {
					t.Fatalf("candidate details omitted old address %s: %s", actor.Binding.Slot, detail)
				}
			}
			for _, doc := range documents {
				var got struct {
					Code, Error  string
					Registration sessionsync.Registration
				}
				if err := json.Unmarshal([]byte(upgradeCLI(t, ctx, fleet, "--session-sync", "lookup", doc.File, "--json")), &got); err != nil {
					t.Fatal(err)
				}
				if got.Code != "ok" || !reflect.DeepEqual(got.Registration, doc) {
					t.Fatalf("candidate changed old registry identity for %s", doc.File)
				}
				var read struct {
					Code, Error string
					Read        sessionsync.ReadResult
				}
				if err := json.Unmarshal([]byte(upgradeCLI(t, ctx, fleet, "--session-sync", "read", "--all", doc.File, "--json")), &read); err != nil {
					t.Fatal(err)
				}
				if read.Code != "ok" || !strings.Contains(read.Read.Body, "Upgrade") {
					t.Fatalf("candidate did not read old document: %+v", read)
				}
			}
			for n := 0; n <= 1; n++ {
				slot := fmt.Sprintf("fixture:%d", n)
				t.Logf("upgrade phase: candidate explicit resume %s", slot)
				var response couchmessage.Response
				if err := json.Unmarshal([]byte(upgradeCLI(t, ctx, fleet, "--resume", slot, "--json")), &response); err != nil {
					t.Fatal(err)
				}
				if response.Code != "ok" || response.Operation == nil {
					t.Fatalf("candidate resume %s refused: %+v", slot, response)
				}
				id := response.Operation.ID
				if err := waitFleet(ctx, func() (bool, error) {
					result, err := fleet.ActorCLI(ctx, 0, "--operation-status", id, "--json")
					if err != nil {
						return false, err
					}
					var status couchmessage.Response
					if err := json.Unmarshal([]byte(result.Stdout), &status); err != nil {
						return false, err
					}
					if result.ExitCode != 0 || status.Operation == nil {
						return false, fmt.Errorf("resume status unavailable: %s %s", result.Stdout, result.Stderr)
					}
					if status.Operation.Status.Terminal() && status.Operation.Status != couchmessage.ReceiptSucceeded {
						return false, fmt.Errorf("resume %s: %+v", slot, status.Operation)
					}
					return status.Operation.Status == couchmessage.ReceiptSucceeded, nil
				}); err != nil {
					t.Fatalf("candidate explicit resume: %v", err)
				}
				// Open the real switcher, choose the old slot and attach through its
				// ordinary Enter gesture, then require native client evidence.
				upgradeSelect(t, ctx, fleet, before[slot].Binding.Tag)
				forceFleetWaitScreen(t, ctx, fleet, "upgrade-"+slot)
				upgradeClients(t, ctx, fleet, before[slot].Binding.Session, true)
			}
			if err := compareUpgradeCohort(before, snapshotUpgrade(t, ctx, fleet)); err != nil {
				t.Fatal(err)
			}
			t.Logf("upgrade qualified: agent=%s slots=2 public-adoption-list-resume-attach=PASS old-registry-parent-side=PASS identities-sockets-preserved=PASS", kind)
		})
		if !caseCleaned {
			t.Fatal("personality series stopped after uncertain cleanup")
		}
		if !casePassed {
			break
		}
	}
}

func upgradeBuild(t *testing.T, source, scratch string) testcouch.Binaries {
	t.Helper()
	if !filepath.IsAbs(source) || filepath.Clean(source) != source {
		t.Fatalf("explicit clean absolute upgrade source required: %q", source)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	git := func(args ...string) string {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", source}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("upgrade source %s: %v %s", source, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	head := git("rev-parse", "HEAD")
	if dirty := git("status", "--porcelain"); dirty != "" {
		t.Fatalf("upgrade source must be committed and clean: %s\n%s", source, dirty)
	}
	owner, err := testcouch.BuildOwned(ctx, testcouch.BuildConfig{Source: source, ScratchParent: scratch})
	if owner != nil {
		t.Cleanup(func() {
			cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
			defer stop()
			if err := owner.Close(cleanup); err != nil {
				t.Error(err)
			}
		})
	}
	if err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{"pair": owner.Binaries.Pair, "couch": owner.Binaries.Couch, "helper": owner.Binaries.Helper} {
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, io.LimitReader(file, 512<<20+1))
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil {
			t.Fatalf("candidate identity: %v %v", copyErr, closeErr)
		}
		info, err := buildinfo.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		settings := map[string]string{}
		for _, setting := range info.Settings {
			settings[setting.Key] = setting.Value
		}
		if settings["vcs.revision"] != head || settings["vcs.modified"] != "false" {
			t.Fatalf("%s build does not match committed source %s: %v", name, head, settings)
		}
		t.Logf("upgrade build: source=%s head=%s binary=%s sha256=%x vcs.modified=false", source, head, name, hash.Sum(nil))
	}
	return owner.Binaries
}

type upgradeActor struct {
	Binding couchmessage.Binding
	Ready   agentfixture.Ready
	Socket  string
}

func snapshotUpgrade(t *testing.T, ctx context.Context, fleet *testcouch.Fleet) map[string]upgradeActor {
	t.Helper()
	actors, err := fleet.Actors(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(actors.Actors) != 2 {
		t.Fatalf("upgrade needs both old actors: %d", len(actors.Actors))
	}
	out := map[string]upgradeActor{}
	for n := 0; n <= 1; n++ {
		slot := fmt.Sprintf("fixture:%d", n)
		ready, err := fleet.Agent(n)
		if err != nil {
			t.Fatal(err)
		}
		for _, actor := range actors.Actors {
			if actor.Binding.Slot != slot {
				continue
			}
			if err := actor.Binding.Validate(); err != nil {
				t.Fatal(err)
			}
			if _, exists := out[slot]; exists {
				t.Fatalf("duplicate old slot %s", slot)
			}
			path := filepath.Join(fleet.Roots.Mutable["sockets"], "contract_version_1", actor.Binding.Session)
			socket, err := os.Lstat(path)
			if err != nil || socket.Mode()&os.ModeSocket == 0 {
				t.Fatalf("old native session socket: %s %v", path, err)
			}
			identity, ok := socket.Sys().(*syscall.Stat_t)
			if !ok {
				t.Fatal("native socket file identity unavailable")
			}
			out[slot] = upgradeActor{Binding: actor.Binding, Ready: ready, Socket: fmt.Sprintf("%d:%d", identity.Dev, identity.Ino)}
		}
		if _, ok := out[slot]; !ok {
			t.Fatalf("missing old slot %s", slot)
		}
	}
	// This path is the captured pre-#866 compatibility contract. Observe only:
	// deleting or seeding markers would cease to exercise an actual old writer.
	markerRoot := filepath.Join(fleet.Roots.Mutable["home"], ".cache", "pair", "zellij-ready-v1")
	if entries, err := os.ReadDir(markerRoot); !os.IsNotExist(err) && (err != nil || len(entries) != 0) {
		t.Fatalf("legacy marker state changed: %s %v", markerRoot, err)
	}
	return out
}

func compareUpgradeCohort(before, after map[string]upgradeActor) error {
	if len(before) != 2 || len(after) != len(before) {
		return fmt.Errorf("cohort size changed")
	}
	for slot, old := range before {
		current, exists := after[slot]
		if !exists || old.Binding != current.Binding || !reflect.DeepEqual(old.Ready, current.Ready) || old.Socket == "" || old.Socket != current.Socket {
			return fmt.Errorf("old actor/session identity changed at %s", slot)
		}
	}
	return nil
}

func upgradeCLI(t *testing.T, ctx context.Context, fleet *testcouch.Fleet, args ...string) string {
	t.Helper()
	result, err := fleet.ActorCLI(ctx, 0, args...)
	if err != nil || result.ExitCode != 0 || result.Truncated {
		t.Fatalf("upgrade public %v: exit=%d err=%v stdout=%s stderr=%s", args, result.ExitCode, err, result.Stdout, result.Stderr)
	}
	return result.Stdout
}

func upgradeDocuments(t *testing.T, ctx context.Context, fleet *testcouch.Fleet) []sessionsync.Registration {
	t.Helper()
	documents := []struct{ path, body string }{
		{"upgrade.md", "---\ntype: prose\n---\n# Upgrade parent\n"},
		{"upgrade.side/child.md", "---\ntype: side-doc\nparent: ../upgrade.md\n---\n# Upgrade child\n"},
	}
	var out []sessionsync.Registration
	for _, doc := range documents {
		file := filepath.Join(fleet.Workspaces.Primary, doc.path)
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(doc.body), 0600); err != nil {
			t.Fatal(err)
		}
		var result struct {
			Code, Error  string
			Registration sessionsync.Registration
		}
		if err := json.Unmarshal([]byte(upgradeCLI(t, ctx, fleet, "--session-sync", "register", file, "--owner", "fixture:0", "--json")), &result); err != nil {
			t.Fatal(err)
		}
		if result.Code != "ok" || result.Registration.ID == "" || result.Registration.File != file {
			t.Fatalf("old public document writer: %+v", result)
		}
		out = append(out, result.Registration)
	}
	return out
}

func upgradeClients(t *testing.T, ctx context.Context, fleet *testcouch.Fleet, session string, wantAttached bool) {
	t.Helper()
	query := zellijquery.Source{Path: fleet.Binaries.Zellij, Env: fleet.Environment}
	if err := waitFleet(ctx, func() (bool, error) {
		raw, err := query.Output(ctx, "--session", session, "action", "list-clients")
		if err != nil {
			return false, err
		}
		clients, err := zellijreply.ParseClients(raw)
		if err != nil {
			return false, err
		}
		return (len(clients) > 0) == wantAttached, nil
	}); err != nil {
		t.Fatalf("native client evidence attached=%v session=%s: %v", wantAttached, session, err)
	}
}

func upgradeSelect(t *testing.T, ctx context.Context, fleet *testcouch.Fleet, slot string) {
	t.Helper()
	wakeFleetOpenSwitcher(t, ctx, fleet)
	if err := fleet.Input([]byte(slot)); err != nil {
		t.Fatal(err)
	}
	forceFleetPress(t, fleet, "\r", 13, 1)
	if err := waitFleet(ctx, func() (bool, error) {
		return !strings.Contains(fleet.Screen(), "Ctrl+Y copy last notice / selected error"), nil
	}); err != nil {
		t.Fatalf("physical selection of %s stayed in switcher: %v; screen=%s", slot, err, fleet.Screen())
	}
}

func TestUpgradeCohortComparisonRejectsChangedOldIdentity(t *testing.T) {
	for _, scenario := range []string{"same", "missing", "extra", "binding", "pid", "generation", "native-id", "socket"} {
		t.Run(scenario, func(t *testing.T) {
			before := map[string]upgradeActor{}
			after := map[string]upgradeActor{}
			for n := 0; n <= 1; n++ {
				slot := fmt.Sprintf("fixture:%d", n)
				actor := upgradeActor{Binding: couchmessage.Binding{Slot: slot, Nonce: "old"}, Ready: agentfixture.Ready{PID: n + 10, Generation: "old", NativeID: slot}, Socket: slot}
				before[slot], after[slot] = actor, actor
			}
			actor := after["fixture:1"]
			switch scenario {
			case "missing":
				delete(after, "fixture:0")
			case "extra":
				after["fixture:2"] = actor
			case "binding":
				actor.Binding.Nonce = "new"
			case "pid":
				actor.Ready.PID++
			case "generation":
				actor.Ready.Generation = "new"
			case "native-id":
				actor.Ready.NativeID = "new"
			case "socket":
				actor.Socket = "new"
			}
			after["fixture:1"] = actor
			if err := compareUpgradeCohort(before, after); (err == nil) != (scenario == "same") {
				t.Fatalf("comparison for %s: %v", scenario, err)
			}
		})
	}
}
