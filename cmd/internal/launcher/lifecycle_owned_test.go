package launcher

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func deleteOwnedSession(t *testing.T, rt OSRuntime, ctx context.Context, owner SessionOwnerObservation) error {
	t.Helper()
	deleter, ok := any(rt).(interface {
		DeleteOwnedSessionContext(context.Context, SessionOwnerObservation) error
	})
	if !ok {
		t.Fatal("OSRuntime lacks deletion bound to prior ownership proof")
	}
	return deleter.DeleteOwnedSessionContext(ctx, owner)
}

func TestDeleteOwnedSessionRetainsOriginalAuthorization(t *testing.T) {
	a := SessionServerIdentity{PID: 1, Identity: "original", Session: "work", Socket: "/private/work"}
	b := SessionServerIdentity{PID: 2, Identity: "replacement", Session: "work", Socket: "/private/work"}
	for _, kind := range []string{"exact", "independently absent", "replacement before first census", "contested before first census", "same PID reused", "socket changed", "replacement during loop", "new server after absence"} {
		t.Run(kind, func(t *testing.T) {
			o := &replacingQuiescenceOps{servers: []SessionServerIdentity{a}}
			var wantKills []SessionServerIdentity
			wantErr := true
			switch kind {
			case "exact":
				wantKills, wantErr = []SessionServerIdentity{a}, false
			case "independently absent":
				o.servers, wantErr = nil, false
			case "replacement before first census":
				o.servers = []SessionServerIdentity{b}
			case "contested before first census":
				o.servers = []SessionServerIdentity{a, b}
			case "same PID reused":
				reused := b
				reused.PID = a.PID
				o.servers = []SessionServerIdentity{reused}
			case "socket changed":
				changed := a
				changed.Socket = "/other/work"
				o.servers = []SessionServerIdentity{changed}
			case "replacement during loop":
				wantKills = []SessionServerIdentity{a}
				o.beforeObserve = func(o *replacingQuiescenceOps) {
					if o.observations == 2 {
						o.servers = []SessionServerIdentity{b}
					}
				}
			case "new server after absence":
				o.servers = nil
				o.beforeObserve = func(o *replacingQuiescenceOps) {
					if o.observations == 2 {
						o.servers = []SessionServerIdentity{b}
					}
				}
			}
			rt := OSRuntime{sessionQuiescence: o, sessionQuiesceWait: time.Second, sessionQuiescePoll: time.Nanosecond}
			err := deleteOwnedSession(t, rt, t.Context(), SessionOwnerObservation{State: SessionOwnerOwned, Name: "work", Server: a})
			if (err != nil) != wantErr {
				t.Fatalf("error=%v wantErr=%v", err, wantErr)
			}
			if !reflect.DeepEqual(o.killed, wantKills) {
				t.Fatalf("signals=%+v want %+v", o.killed, wantKills)
			}
			if !wantErr && o.observations < 2 {
				t.Fatalf("absence not confirmed: %d observations", o.observations)
			}
		})
	}
}

func TestDeleteOwnedSessionRefusesInvalidProofBeforeObservations(t *testing.T) {
	valid := SessionOwnerObservation{State: SessionOwnerOwned, Name: "work", Server: SessionServerIdentity{PID: 1, Identity: "original", Session: "work", Socket: "/private/work"}}
	for _, kind := range []string{"unknown", "absent", "foreign", "orphaned", "zero PID", "negative PID", "missing identity", "wrong session", "missing name", "relative socket", "unclean socket", "wrong socket leaf", "root session"} {
		t.Run(kind, func(t *testing.T) {
			owner := valid
			switch kind {
			case "unknown":
				owner.State = SessionOwnerUnknown
			case "absent":
				owner.State = SessionOwnerAbsent
			case "foreign":
				owner.State = SessionOwnerForeign
			case "orphaned":
				owner.State = SessionOwnerOrphaned
			case "zero PID":
				owner.Server.PID = 0
			case "negative PID":
				owner.Server.PID = -1
			case "missing identity":
				owner.Server.Identity = ""
			case "wrong session":
				owner.Server.Session = "other"
			case "missing name":
				owner.Name = ""
			case "relative socket":
				owner.Server.Socket = "private/work"
			case "unclean socket":
				owner.Server.Socket = "/private/../private/work"
			case "wrong socket leaf":
				owner.Server.Socket = "/private/other"
			case "root session":
				owner.Name, owner.Server.Session, owner.Server.Socket = "/", "/", "/"
			}
			o := &replacingQuiescenceOps{servers: []SessionServerIdentity{valid.Server}}
			err := deleteOwnedSession(t, OSRuntime{sessionQuiescence: o}, t.Context(), owner)
			if err == nil || o.observations != 0 || len(o.killed) != 0 {
				t.Fatalf("invalid proof reached IO: err=%v observations=%d kills=%v", err, o.observations, o.killed)
			}
		})
	}
}

func TestDeleteOwnedSessionHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	o := &deadlineCaptureOps{}
	owner := SessionOwnerObservation{State: SessionOwnerOwned, Name: "work", Server: SessionServerIdentity{PID: 1, Identity: "original", Session: "work", Socket: "/private/work"}}
	if err := deleteOwnedSession(t, OSRuntime{sessionQuiescence: o}, ctx, owner); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	if o.calls != 0 {
		t.Fatalf("canceled deletion observed %d times", o.calls)
	}
}
