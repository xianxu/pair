package launcher

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

func TestLegacyNativeSourceRequiresHeldOwnerTranscript(t *testing.T) {
	const id = "019d1111-1111-7111-8111-111111111111"
	for _, mode := range []string{"direct", "node-engine", "same-id-other-owner", "no-held-file", "wrong-held-id", "tool-descendant", "wrong-parent", "modern-mismatch", "engine-reused", "held-file-changed", "held-read-fails"} {
		t.Run(mode, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
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
				return []string{filepath.Join(home, ".codex", "sessions", "unrelated.jsonl"), root}, nil
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
