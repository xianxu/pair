package launcher

import (
	"context"
	"errors"
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
	deleteErr       error
	killErr         error
	deleteCalls     int
	killed          []int
	reRegistrates   int
}

func (f *statefulSessionQuiescenceOps) SessionPresent(context.Context, string) (bool, error) {
	if f.probeErr != nil {
		return false, f.probeErr
	}
	return f.present, nil
}

func (f *statefulSessionQuiescenceOps) SessionServers(_ context.Context, session string) ([]sessionServerIdentity, error) {
	if f.serverAlive {
		identity := f.serverIdentity
		if identity == "" {
			identity = "start-a"
		}
		return []sessionServerIdentity{{PID: f.serverPID, Identity: identity, Session: session}}, nil
	}
	return nil, nil
}

func (f *statefulSessionQuiescenceOps) DeleteSessionRecord(context.Context, string) error {
	f.deleteCalls++
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.present = false
	if f.serverAlive {
		f.present = true
		f.reRegistrates++
	}
	return nil
}

func (f *statefulSessionQuiescenceOps) KillServer(server sessionServerIdentity) error {
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
			if err := ops.KillServer(sessionServerIdentity{PID: 4242, Identity: "zellij-start", Session: "pair-work"}); err != nil {
				t.Fatal(err)
			}
			if killed != tc.wantKill {
				t.Fatalf("killed = %v, want %v", killed, tc.wantKill)
			}
		})
	}
}

func TestOSRuntimeDeleteSessionProvesReRegisteringServerAbsent(t *testing.T) {
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
	if ops.reRegistrates != 1 || ops.deleteCalls < 2 || !reflect.DeepEqual(ops.killed, []int{4242}) {
		t.Fatalf("re-registration lifecycle was not observed: %+v", ops)
	}
}

func TestDeleteSessionFailsClosedWhenAbsenceCannotBeObserved(t *testing.T) {
	for _, tc := range []struct {
		name string
		ops  *statefulSessionQuiescenceOps
	}{
		{name: "probe error", ops: &statefulSessionQuiescenceOps{probeErr: errors.New("zellij query unavailable")}},
		{name: "delete error", ops: &statefulSessionQuiescenceOps{present: true, deleteErr: errors.New("delete refused")}},
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
func (o *deadlineCaptureOps) SessionServers(ctx context.Context, _ string) ([]sessionServerIdentity, error) {
	o.capture(ctx)
	return nil, nil
}
func (o *deadlineCaptureOps) DeleteSessionRecord(ctx context.Context, _ string) error {
	o.capture(ctx)
	return nil
}
func (*deadlineCaptureOps) KillServer(sessionServerIdentity) error { return nil }

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
