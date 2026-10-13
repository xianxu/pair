package couchcore

import (
	"context"
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

type contestedDetachedOwner struct{ *sessionOwnerWorld }

func (w contestedDetachedOwner) SessionServers(ctx context.Context, name string) ([]launcher.SessionServerIdentity, error) {
	servers, err := w.sessionOwnerWorld.SessionServers(ctx, name)
	other := servers[0]
	other.PID++
	other.Identity = "other"
	return append(servers, other), err
}

func TestDetachedSessionsRequireAttestedUsableTerminal(t *testing.T) {
	for _, kind := range []string{"healthy", "legacy", "empty agent", "contested", "wrong agent", "unreadable processes"} {
		t.Run(kind, func(t *testing.T) {
			data := t.TempDir()
			const name = "📁repo-a"
			indexSession(t, data, addressA, name)
			checker, _ := sandboxedChecker(t, data, map[string]string{name: "detached"})
			reads, calls := 0, 0
			checker.ProcessTable = func() ([]procutil.Process, error) {
				reads++
				if kind == "unreadable processes" {
					return nil, errors.New("unavailable")
				}
				return []procutil.Process{{PID: 99}}, nil
			}
			checker.TerminalUsability = func(server launcher.SessionServerIdentity, root, scope, tag string, table []procutil.Process) launcher.TerminalSafety {
				calls++
				if server.PID != 42 || root != data || scope != addressA.RepoScope || tag != string(addressA.Tag) || len(table) != 1 {
					t.Fatalf("wrong proof inputs: %+v %s %s %s %v", server, root, scope, tag, table)
				}
				policy := artifactpath.SessionLaunchPolicy{Agent: "claude"}
				if kind == "wrong agent" {
					policy.Agent = "codex"
				}
				return launcher.TerminalSafety{Usable: kind != "legacy" && kind != "empty agent", Policy: policy}
			}
			if kind == "contested" {
				checker.OwnerProbe = launcher.SessionOwnerProbe{IO: contestedDetachedOwner{&sessionOwnerWorld{root: data, owners: map[string]ThreadAddress{name: addressA}, generation: "first"}}}
			}
			got, err := checker.DetachedSessions(t.Context(), []DetachedCandidate{{Address: addressA, Agent: "claude", SessionName: name}})
			if kind == "healthy" {
				if err != nil || len(got) != 1 || reads != 1 || calls != 1 {
					t.Fatalf("healthy proof: got=%v err=%v reads=%d calls=%d", got, err, reads, calls)
				}
			} else if len(got) != 0 {
				t.Fatalf("unsafe %s admitted: %v err=%v", kind, got, err)
			}
			if (kind == "contested" || kind == "unreadable processes") && calls != 0 {
				t.Fatal("usability observed without unique ownership/process proof")
			}
		})
	}
}
