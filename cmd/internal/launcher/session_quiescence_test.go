package launcher

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

type statefulSessionQuiescenceOps struct {
	present         bool
	serverPID       int
	serverIdentity  string
	serverAlive     bool
	reuseBeforeKill bool
	unrelatedKilled bool
	probeErr        error
	killErr         error
	killed          []int
}

func (f *statefulSessionQuiescenceOps) SessionPresent(context.Context, string) (bool, error) {
	if f.probeErr != nil {
		return false, f.probeErr
	}
	return f.present, nil
}

func (f *statefulSessionQuiescenceOps) SessionServers(_ context.Context, session string) ([]SessionServerIdentity, error) {
	if f.serverAlive {
		identity := f.serverIdentity
		if identity == "" {
			identity = "start-a"
		}
		return []SessionServerIdentity{{PID: f.serverPID, Identity: identity, Session: session, Socket: "/tmp/" + session}}, nil
	}
	return nil, nil
}

func (f *statefulSessionQuiescenceOps) KillServer(server SessionServerIdentity) error {
	f.killed = append(f.killed, server.PID)
	if f.killErr != nil {
		return f.killErr
	}
	if f.reuseBeforeKill {
		f.serverAlive = false
		f.unrelatedKilled = false
		return nil
	}
	if server.PID == f.serverPID {
		f.serverAlive = false
		f.present = false
	}
	return nil
}

func TestOSSessionKillReauthorizesExactProcessIdentity(t *testing.T) {
	for _, tc := range []struct {
		name       string
		identities []string
		command    string
		wantKill   bool
	}{
		{name: "PID reused", identities: []string{"reused-start"}, command: "/opt/bin/zellij --server /tmp/pair-work"},
		{name: "process execed away", identities: []string{"zellij-start"}, command: "/usr/bin/unrelated --worker"},
		{name: "reuse during reauthorization", identities: []string{"zellij-start", "reused-start"}, command: "/opt/bin/zellij --server /tmp/pair-work"},
		{name: "socket replaced", identities: []string{"zellij-start"}, command: "/opt/bin/zellij --server /other/pair-work"},
		{name: "exact incarnation", identities: []string{"zellij-start"}, command: "/opt/bin/zellij --server /tmp/pair-work", wantKill: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			killed := false
			identityCall := 0
			ops := osSessionQuiescenceOps{
				processIdentity: func(string) string {
					index := identityCall
					identityCall++
					if index >= len(tc.identities) {
						index = len(tc.identities) - 1
					}
					return tc.identities[index]
				},
				processCommand: func(string) string { return tc.command },
				killProcess:    func(int) error { killed = true; return nil },
			}
			if err := ops.KillServer(SessionServerIdentity{PID: 4242, Identity: "zellij-start", Session: "pair-work", Socket: "/tmp/pair-work"}); err != nil {
				t.Fatal(err)
			}
			if killed != tc.wantKill {
				t.Fatalf("killed = %v, want %v", killed, tc.wantKill)
			}
		})
	}
}

func TestOSRuntimeDeleteSessionKillsExactServerWithoutDeletingLayout(t *testing.T) {
	t.Setenv("ZELLIJ_SOCKET_DIR", filepath.Join(t.TempDir(), "absent"))
	ops := &statefulSessionQuiescenceOps{present: true, serverPID: 4242, serverAlive: true}
	rt := OSRuntime{
		sessionQuiescence:  ops,
		sessionQuiesceWait: 100 * time.Millisecond,
		sessionQuiescePoll: time.Millisecond,
	}
	if err := rt.DeleteSession("pair-work"); err != nil {
		t.Fatal(err)
	}
	if ops.present || ops.serverAlive {
		t.Fatalf("successful delete left state: %+v", ops)
	}
	if !reflect.DeepEqual(ops.killed, []int{4242}) {
		t.Fatalf("quiescence missed exact server: %+v", ops)
	}
}

func TestDeleteSessionFailsClosedWhenAbsenceCannotBeObserved(t *testing.T) {
	for _, tc := range []struct {
		name string
		ops  *statefulSessionQuiescenceOps
	}{
		{name: "probe error", ops: &statefulSessionQuiescenceOps{probeErr: errors.New("zellij query unavailable")}},
		{name: "present without attributable server", ops: &statefulSessionQuiescenceOps{present: true}},
		{name: "kill error", ops: &statefulSessionQuiescenceOps{serverPID: 4242, serverAlive: true, killErr: errors.New("kill refused")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := quiesceZellijSession(context.Background(), "pair-work", tc.ops, 10*time.Millisecond, time.Millisecond)
			if err == nil {
				t.Fatal("unobservable absence reported success")
			}
		})
	}
}

type deadlineCaptureOps struct {
	calls     int
	deadlines []time.Time
}

func (o *deadlineCaptureOps) capture(ctx context.Context) {
	o.calls++
	deadline, _ := ctx.Deadline()
	o.deadlines = append(o.deadlines, deadline)
}
func (o *deadlineCaptureOps) SessionPresent(ctx context.Context, _ string) (bool, error) {
	o.capture(ctx)
	return false, nil
}
func (o *deadlineCaptureOps) SessionServers(ctx context.Context, _ string) ([]SessionServerIdentity, error) {
	o.capture(ctx)
	return nil, nil
}
func (*deadlineCaptureOps) KillServer(SessionServerIdentity) error { return nil }

func TestCleanupDeadlinePropagation(t *testing.T) {
	parentDeadline := time.Now().Add(time.Hour)
	parent, cancel := context.WithDeadline(context.Background(), parentDeadline)
	defer cancel()
	ops := &deadlineCaptureOps{}
	if err := quiesceZellijSession(parent, "pair-work", ops, zjTimeout, time.Nanosecond); err != nil {
		t.Fatal(err)
	}
	if ops.calls == 0 {
		t.Fatal("quiescence made no observations")
	}
	for _, deadline := range ops.deadlines {
		if !deadline.Before(parentDeadline) {
			t.Fatalf("child deadline %v did not apply inner %v bound before parent %v", deadline, zjTimeout, parentDeadline)
		}
	}

	cancelled, stop := context.WithCancel(context.Background())
	stop()
	cancelledOps := &deadlineCaptureOps{}
	if err := quiesceZellijSession(cancelled, "pair-work", cancelledOps, zjTimeout, time.Nanosecond); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled parent error=%v", err)
	}
	if cancelledOps.calls != 0 {
		t.Fatalf("cancelled parent reached observations: %d", cancelledOps.calls)
	}
}

func TestZellijServerPIDsMatchOnlyExactSession(t *testing.T) {
	raw := `
  41 zellij --server /tmp/zellij/pair-work
  42 zellij --server /tmp/zellij/pair-work-neighbor
  43 sh -c zellij --server /tmp/zellij/pair-work
  44 /opt/bin/zellij --server /tmp/zellij/pair-work
`
	if got := zellijServerPIDs(raw, "pair-work"); !reflect.DeepEqual(got, []int{41, 44}) {
		t.Fatalf("exact server pids = %v", got)
	}
}

func TestOSSessionQuiescenceEmptyInventory(t *testing.T) {
	for _, tc := range []struct {
		name      string
		wantError bool
	}{
		{"empty", false},
		{"unknown", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			stub := "#!/bin/sh\n" + zellijtest.ShellCase(t, "launcher-quiescence-unknown")
			if !tc.wantError {
				stub = "#!/bin/sh\n" + zellijtest.ShellCase(t, "launcher-quiescence-no-sessions")
			}
			if err := os.WriteFile(filepath.Join(dir, "zellij"), []byte(stub), 0755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir)
			present, err := (osSessionQuiescenceOps{}).SessionPresent(context.Background(), "pair-fixture")
			if present || (err != nil) != tc.wantError {
				t.Fatalf("present=%v err=%v, want absent and error=%v", present, err, tc.wantError)
			}
		})
	}
}

type replacingQuiescenceOps struct {
	servers       []SessionServerIdentity
	observations  int
	beforeObserve func(*replacingQuiescenceOps)
	killed        []SessionServerIdentity
}

func (o *replacingQuiescenceOps) SessionPresent(context.Context, string) (bool, error) {
	return len(o.servers) > 0, nil
}
func (o *replacingQuiescenceOps) SessionServers(context.Context, string) ([]SessionServerIdentity, error) {
	o.observations++
	if o.beforeObserve != nil {
		o.beforeObserve(o)
	}
	return append([]SessionServerIdentity(nil), o.servers...), nil
}
func (o *replacingQuiescenceOps) KillServer(s SessionServerIdentity) error {
	o.killed = append(o.killed, s)
	for i, x := range o.servers {
		if x == s {
			o.servers = append(o.servers[:i:i], o.servers[i+1:]...)
			break
		}
	}
	return nil
}
func TestQuiescenceRejectsContestedAndReplacementBeforeEffects(t *testing.T) {
	a := SessionServerIdentity{PID: 1, Identity: "a", Session: "work", Socket: "/tmp/work"}
	b := SessionServerIdentity{PID: 2, Identity: "b", Session: "work", Socket: "/tmp/work"}
	for _, kind := range []string{"contested", "replacement", "same PID new start", "socket replaced", "unknown identity"} {
		t.Run(kind, func(t *testing.T) {
			o := &replacingQuiescenceOps{servers: []SessionServerIdentity{a}}
			switch kind {
			case "contested":
				o.servers = append(o.servers, b)
			default:
				o.beforeObserve = func(o *replacingQuiescenceOps) {
					if o.observations == 2 {
						replacement := b
						if kind == "same PID new start" {
							replacement.PID = a.PID
						}
						if kind == "socket replaced" {
							replacement = a
							replacement.Socket = "/other/work"
						}
						if kind == "unknown identity" {
							replacement = a
							replacement.Identity = ""
						}
						o.servers = []SessionServerIdentity{replacement}
					}
				}
			}
			if err := quiesceZellijSession(context.Background(), "work", o, time.Second, time.Nanosecond); err == nil {
				t.Fatal("unsafe quiescence succeeded")
			}
			if len(o.killed) != 0 {
				t.Fatalf("unauthorized effects: kills=%v", o.killed)
			}
		})
	}
}

func TestAuthorizedQuiescenceAllowsOnlyAttributedSnapshot(t *testing.T) {
	a := SessionServerIdentity{PID: 1, Identity: "a", Session: "work", Socket: "/tmp/work"}
	b := SessionServerIdentity{PID: 2, Identity: "b", Session: "work", Socket: "/other/work"}
	o := &replacingQuiescenceOps{servers: []SessionServerIdentity{a, b}}
	if err := quiesceAuthorizedSession(t.Context(), "work", []SessionServerIdentity{a, b}, o, time.Second, time.Nanosecond); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(o.killed, []SessionServerIdentity{a, b}) {
		t.Fatalf("effects: kills=%v", o.killed)
	}
}

func TestQuiescenceRefusesReplacementAfterAuthorizedKill(t *testing.T) {
	a := SessionServerIdentity{PID: 1, Identity: "a", Session: "work", Socket: "/tmp/work"}
	b := SessionServerIdentity{PID: 2, Identity: "b", Session: "work", Socket: "/tmp/work"}
	o := &replacingQuiescenceOps{servers: []SessionServerIdentity{a}}
	o.beforeObserve = func(o *replacingQuiescenceOps) {
		if o.observations == 3 {
			o.servers = []SessionServerIdentity{b}
		}
	}
	if err := quiesceZellijSession(t.Context(), "work", o, time.Second, time.Nanosecond); err == nil {
		t.Fatal("replacement must refuse")
	}
	if !reflect.DeepEqual(o.killed, []SessionServerIdentity{a}) {
		t.Fatalf("effects: kills=%v", o.killed)
	}
}

func TestOSSessionQuiescenceRetainsSavedLayout(t *testing.T) {
	dir := t.TempDir()
	stub := "#!/bin/sh\n" + zellijtest.ShellCase(t, "launcher-quiescence-exited")
	if err := os.WriteFile(filepath.Join(dir, "zellij"), []byte(stub), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	present, err := (osSessionQuiescenceOps{}).SessionPresent(t.Context(), "pair-fixture")
	if err != nil || present {
		t.Fatalf("saved layout counted as live: present=%v err=%v", present, err)
	}
}
