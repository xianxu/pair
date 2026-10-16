package launcher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func TestLegacyNativeSourceRequiresHeldOwnerTranscript(t *testing.T) {
	const id = "019d1111-1111-7111-8111-111111111111"
	for _, mode := range []string{"direct", "node-engine", "same-id-other-owner", "no-held-file", "wrong-held-id", "tool-descendant", "wrong-parent", "modern-mismatch", "engine-reused", "held-file-changed", "held-read-fails"} {
		t.Run(mode, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("CODEX_HOME", "")
			w, check := nativeOwnerFixture(t, "codex", id)
			check.NativeID, check.Tag = id, "other"
			check.SourcePID, check.SourceIdentity = 7, "helper"
			check.SourceSession, check.SourceNonce = "session", "nonce"
			w.ready.AgentIdentity = ""
			s := w.snapshots[0]
			s.Processes[0].PPID = 7
			s.Processes[0].Name = "pair-wrap"
			s.Processes[1].Name = "codex"
			s.Processes = append(s.Processes, procutil.Process{PID: 7, Name: "pair", Identity: "helper"})
			s.Commands[7] = "pair resume other"
			root := filepath.Join(home, ".codex", "sessions", "2026", "10", "15", "rollout-2026-10-15T00-00-00-"+id+".jsonl")
			holder := "43"
			if mode != "direct" && mode != "modern-mismatch" {
				s.Processes[1].Name = "node"
				s.Commands[43] = "node /usr/bin/codex"
				s.Processes = append(s.Processes, procutil.Process{PID: 44, PPID: 43, Name: "codex", Identity: "native-engine"})
				s.Commands[44] = "/bin/codex"
				holder = "44"
			}
			switch mode {
			case "same-id-other-owner":
				check.SourcePID = 0
			case "wrong-held-id":
				root = filepath.Join(home, ".codex", "sessions", "rollout-2026-10-15T00-00-00-019d2222-2222-7222-8222-222222222222.jsonl")
			case "tool-descendant":
				s.Processes[len(s.Processes)-1].Name = "bash"
				s.Commands[44] = "bash -c inspect"
			case "wrong-parent":
				s.Processes[1].PPID = 99
			case "modern-mismatch":
				w.ready.AgentIdentity = "wrong-generation"
			}
			w.snapshots = []recoveryTerminalSnapshot{s}
			if mode == "engine-reused" {
				next := s
				next.Processes = append([]procutil.Process(nil), s.Processes...)
				next.Processes[len(next.Processes)-1].Identity = "replacement-engine"
				w.snapshots = append(w.snapshots, next)
			}
			calls := 0
			check.heldFiles = func(_ context.Context, pid string) ([]string, error) {
				calls++
				if mode == "held-read-fails" {
					return nil, errors.New("inspection unavailable")
				}
				if pid != holder || mode == "no-held-file" || (mode == "held-file-changed" && w.observations > 1) {
					return nil, nil
				}
				return []string{filepath.Join(home, ".cache", "unrelated.jsonl"), root}, nil
			}
			err := checkRecoveryNativeOwnersWithDiscovery(context.Background(), check, w.Observe, w.Read, orphanOwnerDiscovery(check))
			allowed := mode == "direct" || mode == "node-engine"
			if (err == nil) != allowed {
				t.Fatalf("mode=%s err=%v calls=%d, allowed=%t", mode, err, calls, allowed)
			}
			if mode == "modern-mismatch" && calls != 0 {
				t.Fatal("modern identity mismatch used legacy exception")
			}
			if allowed && (w.observations != 2 || calls < 2) {
				t.Fatalf("legacy exemption was not rechecked: observations=%d reads=%d", w.observations, calls)
			}
		})
	}
}

func TestNativeUnboundUsesCompleteHeldIDSet(t *testing.T) {
	const id = "019d1111-1111-7111-8111-111111111111"
	const other = "019d2222-2222-7222-8222-222222222222"
	for _, legacy := range []bool{false, true} {
		for _, mode := range []string{"different", "collision", "source", "missing", "partial", "changed-set", "conflicting-ledger", "unrecognized-native", "identity-mismatch"} {
			t.Run(map[bool]string{true: "legacy/", false: "modern/"}[legacy]+mode, func(t *testing.T) {
				home := t.TempDir()
				t.Setenv("HOME", home)
				t.Setenv("CODEX_HOME", "")
				w, check := nativeOwnerFixture(t, "codex", id)
				launch := sessionledger.ParseLedger(w.raw).Records[0]
				launch.RequestOrigin, launch.RequestedNativeID = "", ""
				raw, err := sessionledger.EncodeRecord(launch)
				if err != nil {
					t.Fatal(err)
				}
				w.raw = append(raw, '\n')
				if mode == "conflicting-ledger" {
					w.raw = append(w.raw, []byte("malformed\n")...)
				}
				if legacy {
					w.ready.AgentIdentity = ""
				}
				if mode == "identity-mismatch" {
					w.ready.AgentIdentity = "wrong-generation"
				}
				s := w.snapshots[0]
				s.Processes[0].Name, s.Processes[1].Name = "pair-wrap", "codex"
				w.snapshots = []recoveryTerminalSnapshot{s}
				check.NativeID = other
				if mode == "collision" || mode == "source" {
					check.NativeID = id
				}
				if mode == "source" {
					check.Tag, check.SourcePID, check.SourceIdentity = "other", 7, "helper"
					w.snapshots[0].Processes[0].PPID = 7
					w.snapshots[0].Processes = append(w.snapshots[0].Processes, procutil.Process{PID: 7, Name: "pair", Identity: "helper"})
					w.snapshots[0].Commands[7] = "pair resume other"
				}
				check.heldFiles = func(context.Context, string) ([]string, error) {
					if mode == "missing" {
						return nil, nil
					}
					if mode == "partial" {
						return []string{"/partial"}, errors.New("partial observation")
					}
					heldID := id
					if mode == "changed-set" && w.observations > 1 {
						heldID = "019d3333-3333-7333-8333-333333333333"
					}
					paths := []string{filepath.Join(home, ".codex", "sessions", "2026", "10", "15", "rollout-2026-10-15T00-00-00-"+heldID+".jsonl")}
					if mode == "unrecognized-native" {
						paths = append(paths, filepath.Join(home, ".codex", "sessions", "unknown.jsonl"))
					}
					return paths, nil
				}
				err = checkRecoveryNativeOwnersWithDiscovery(context.Background(), check, w.Observe, w.Read, orphanOwnerDiscovery(check))
				allowed := mode == "different" || mode == "source"
				if (err == nil) != allowed {
					t.Fatalf("mode=%s err=%v allowed=%t", mode, err, allowed)
				}
			})
		}
	}
}

func TestLegacyHeldNativeHonorsCodexHome(t *testing.T) {
	for _, mode := range []string{"custom", "default", "invalid", "relative"} {
		t.Run(mode, func(t *testing.T) {
			home, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("HOME", home)
			t.Setenv("CODEX_HOME", "")
			nativeHome := filepath.Join(home, ".codex")
			value := ""
			if mode == "custom" {
				nativeHome = filepath.Join(home, "custom")
				value = nativeHome
			}
			if err := os.MkdirAll(nativeHome, 0700); err != nil {
				t.Fatal(err)
			}
			if mode == "invalid" {
				value = filepath.Join(home, "missing")
			}
			if mode == "relative" {
				value = ".codex"
			}
			t.Setenv("CODEX_HOME", value)
			const id = "019d1111-1111-7111-8111-111111111111"
			check := RecoveryNativeOwnerCheck{Agent: "codex", DataDir: t.TempDir(), heldFiles: func(context.Context, string) ([]string, error) {
				return []string{filepath.Join(nativeHome, "sessions", "2026", "10", "15", "rollout-2026-10-15T00-00-00-"+id+".jsonl")}, nil
			}}
			ids, _, err := checkLegacyHeldNative(context.Background(), check, recoveryTerminalSnapshot{}, procutil.Process{PID: 43, Name: "codex"})
			allowed := mode == "custom" || mode == "default"
			if (err == nil) != allowed || (allowed && (len(ids) != 1 || ids[0] != id)) {
				t.Fatalf("held IDs=%v err=%v", ids, err)
			}
		})
	}
}
