package launcher

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

type nativeOwnerWorld struct {
	snapshots           []recoveryTerminalSnapshot
	raw                 []byte
	ready               readiness.ReadyRecord
	err                 error
	observations, reads int
}

func (w *nativeOwnerWorld) Observe(context.Context, string) (recoveryTerminalSnapshot, error) {
	i := min(w.observations, len(w.snapshots)-1)
	w.observations++
	return w.snapshots[i], nil
}
func (w *nativeOwnerWorld) Read(_ context.Context, path string, _ int64) ([]byte, error) {
	if strings.Contains(path, "agent-ready-") {
		raw, err := readiness.Encode(w.ready)
		return []byte(raw), err
	}
	w.reads++
	return w.raw, w.err
}
func nativeOwnerFixture(t *testing.T, agent, id string) (*nativeOwnerWorld, RecoveryNativeOwnerCheck) {
	t.Helper()
	owner, _ := artifactpath.NewStorageOwner(t.TempDir(), "abcdef1234567890", "other")
	paths, _ := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
	raw, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: owner.RepoScope, Tag: owner.Tag, Agent: agent, RequestOrigin: sessionledger.RequestOriginResume, RequestedNativeID: id})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := recoveryTerminalSnapshot{Processes: []procutil.Process{{PID: 42, PPID: 1, Identity: "wrapper"}, {PID: 43, PPID: 42, Identity: "agent-generation"}}, Commands: map[int]string{43: "codex", 42: "/bin/pair-wrap --scrollback-log " + paths.ScrollbackRaw(agent) + " " + agent}}
	return &nativeOwnerWorld{snapshots: []recoveryTerminalSnapshot{snapshot}, raw: append(raw, '\n'), ready: readiness.ReadyRecord{Tag: owner.Tag, Agent: agent, Session: "session", Nonce: "nonce", PID: 43, AgentIdentity: "agent-generation", LaunchOrdinal: 1}}, RecoveryNativeOwnerCheck{DataDir: owner.DataDir, RepoScope: owner.RepoScope, Tag: "source", Agent: "codex", NativeID: "selected"}
}
func TestRecoveryNativeOwnerGuard(t *testing.T) {
	for _, mode := range []string{"same-native", "different-native", "different-agent", "unreadable", "malformed", "unresolved", "conflicting", "missing-command", "changed-generation"} {
		t.Run(mode, func(t *testing.T) {
			agent, id := "codex", "selected"
			if mode == "different-native" {
				id = "other"
			}
			if mode == "different-agent" {
				agent = "claude"
			}
			w, check := nativeOwnerFixture(t, agent, id)
			switch mode {
			case "unreadable":
				w.err = errors.New("permission denied")
			case "malformed":
				w.raw = []byte("broken\n")
			case "unresolved":
				w.raw = nil
			case "conflicting":
				for _, id := range []string{"selected", "other"} {
					binding, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 1, Kind: sessionledger.RecordBinding, ScopeKey: check.RepoScope, Tag: "other", Agent: "codex", LaunchOrdinal: 1, RootNativeID: id})
					if err != nil {
						t.Fatal(err)
					}
					w.raw = append(w.raw, append(binding, '\n')...)
				}
			case "missing-command":
				delete(w.snapshots[0].Commands, 42)
			case "changed-generation":
				different, _ := nativeOwnerFixture(t, "codex", "other")
				w.raw = different.raw
				w.snapshots = append(w.snapshots, recoveryTerminalSnapshot{Processes: []procutil.Process{{PID: 42, PPID: 1, Identity: "replacement"}}, Commands: map[int]string{42: w.snapshots[0].Commands[42]}})
			}
			err := checkRecoveryNativeOwners(context.Background(), check, w.Observe, w.Read)
			allowed := mode == "different-native" || mode == "different-agent"
			if (err == nil) != allowed {
				t.Fatalf("guard err=%v allowed=%v", err, allowed)
			}
			if mode == "changed-generation" && w.observations != 2 {
				t.Fatal("did not recheck process generation")
			}
			if mode == "different-agent" && w.reads != 0 {
				t.Fatal("read unrelated agent ledger")
			}
		})
	}
}
func TestRecoveryNativeOwnerExcludesOnlyExactSourceGeneration(t *testing.T) {
	w, check := nativeOwnerFixture(t, "codex", "selected")
	check.Tag = "other"
	check.SourcePID = 7
	check.SourceIdentity = "source-helper"
	w.snapshots[0].Processes[0].PPID = 7
	w.snapshots[0].Processes = append(w.snapshots[0].Processes, procutil.Process{PID: 7, PPID: 1, Identity: "source-helper"})
	w.snapshots[0].Commands[7] = "pair resume other"
	if err := checkRecoveryNativeOwners(context.Background(), check, w.Observe, w.Read); err != nil {
		t.Fatal(err)
	}
	check.SourceIdentity = "reused"
	if err := checkRecoveryNativeOwners(context.Background(), check, w.Observe, w.Read); err == nil {
		t.Fatal("excluded replaced source")
	}
	check.SourcePID = 0
	check.SourceIdentity = ""
	if err := checkRecoveryNativeOwners(context.Background(), check, w.Observe, w.Read); err == nil {
		t.Fatal("final gate excluded old source wrapper")
	}
}

func TestRecoveryNativeOwnerExcludesDaemonizedSourceOnlyWithExactReadiness(t *testing.T) {
	w, check := nativeOwnerFixture(t, "codex", "selected")
	check.Tag = "other"
	check.SourcePID = 7
	check.SourceIdentity = "helper"
	check.SourceSession = "source-session"
	check.SourceNonce = "source-nonce"
	w.snapshots[0].Processes = append(w.snapshots[0].Processes, procutil.Process{PID: 7, PPID: 1, Identity: "helper"})
	w.snapshots[0].Commands[7] = "pair resume other"
	w.snapshots[0].Commands[43] = "codex resume selected"
	record := readiness.ReadyRecord{LaunchOrdinal: 1, Tag: "other", Agent: "codex", Session: check.SourceSession, Nonce: check.SourceNonce, PID: 43, AgentIdentity: "agent-generation"}
	read := func(ctx context.Context, path string, max int64) ([]byte, error) {
		if strings.Contains(path, "agent-ready-") {
			raw, err := readiness.Encode(record)
			return []byte(raw), err
		}
		return w.Read(ctx, path, max)
	}
	if err := checkRecoveryNativeOwners(context.Background(), check, w.Observe, read); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"nonce", "agent-generation", "session"} {
		t.Run(mode, func(t *testing.T) {
			changed := record
			switch mode {
			case "nonce":
				record.Nonce = "replacement"
			case "session":
				record.Session = "other-session"
			case "agent-generation":
				record.AgentIdentity = "reused"
			}
			if err := checkRecoveryNativeOwners(context.Background(), check, w.Observe, read); err == nil {
				t.Fatal("excluded stale daemonized source")
			}
			record = changed
		})
	}
}

func TestRecoveryNativeOwnerOldWrapperCannotBorrowNewLaunch(t *testing.T) {
	w, check := nativeOwnerFixture(t, "codex", "selected")
	raw, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: check.RepoScope, Tag: "other", Agent: "codex", RequestOrigin: sessionledger.RequestOriginResume, RequestedNativeID: "different"})
	if err != nil {
		t.Fatal(err)
	}
	w.raw = append(w.raw, append(raw, '\n')...)
	if err := checkRecoveryNativeOwners(context.Background(), check, w.Observe, w.Read); err == nil {
		t.Fatal("old live wrapper hidden by newer launch")
	}
}

func TestRecoveryNativeOwnerRequiresGenerationAssociationEvenForSource(t *testing.T) {
	for _, source := range []bool{false, true} {
		for _, mode := range []string{"missing-ordinal", "wrong-ordinal", "stale-agent", "other-parent", "wrong-tag", "wrong-agent", "missing-ledger", "contradictory-nonce"} {
			t.Run(fmt.Sprintf("source=%v/%s", source, mode), func(t *testing.T) {
				w, check := nativeOwnerFixture(t, "codex", "different")
				if source {
					check.Tag = "other"
					check.SourcePID = 7
					check.SourceIdentity = "helper"
					check.SourceNonce = "nonce"
					check.SourceSession = "session"
					w.snapshots[0].Processes[0].PPID = 7
					w.snapshots[0].Processes = append(w.snapshots[0].Processes, procutil.Process{PID: 7, PPID: 1, Identity: "helper"})
					w.snapshots[0].Commands[7] = "pair resume other"
				}
				switch mode {
				case "missing-ordinal":
					w.ready.LaunchOrdinal = 0
				case "wrong-ordinal":
					w.ready.LaunchOrdinal = 2
				case "stale-agent":
					w.ready.AgentIdentity = "reused"
				case "other-parent":
					w.snapshots[0].Processes[1].PPID = 1
				case "wrong-tag":
					w.ready.Tag = "unrelated"
				case "wrong-agent":
					w.ready.Agent = "claude"
				case "missing-ledger":
					w.raw = nil
				case "contradictory-nonce":
					if !source {
						return
					}
					w.ready.Nonce = "newer"
					check.NativeID = "different"
				}
				if err := checkRecoveryNativeOwners(context.Background(), check, w.Observe, w.Read); err == nil {
					t.Fatal("unproved generation accepted")
				}
			})
		}
	}
}

func TestRecoveryNativeOwnerRechecksReadinessAssociation(t *testing.T) {
	w, check := nativeOwnerFixture(t, "codex", "different")
	reads := 0
	read := func(ctx context.Context, path string, max int64) ([]byte, error) {
		if strings.Contains(path, "agent-ready-") {
			reads++
			if reads == 2 {
				w.ready.Nonce = "replacement"
			}
		}
		return w.Read(ctx, path, max)
	}
	if err := checkRecoveryNativeOwners(context.Background(), check, w.Observe, read); err == nil {
		t.Fatal("readiness changed after process snapshot accepted")
	}
}
