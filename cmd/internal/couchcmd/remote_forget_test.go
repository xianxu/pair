package couchcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/remotesession"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

func TestSessionSyncForgetRefusesOfflineWithoutRemovingRegistration(t *testing.T) {
	rt, file := sessionSyncRT(t)
	syncer := sessionsync.New(rt.namespace.Dir())
	first, err := syncer.Register(file, "ops:0")
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	code, out, errout := runSS(t, rt, "forget", file, "--json")
	if code == 0 || !strings.Contains(out+errout, "start Couch and retry") {
		t.Fatalf("offline forget: %d %s %s", code, out, errout)
	}
	got, err := syncer.Lookup(file)
	if err != nil || got == nil || got.ID != first.Registration.ID {
		t.Fatalf("offline removed registration: %+v %v", got, err)
	}
	after, err := os.ReadFile(file)
	if err != nil || string(after) != string(before) {
		t.Fatal("offline changed file", err)
	}
}

func newForgetOwner(t *testing.T, namespace string) *messageService {
	t.Helper()
	paths, err := artifactpath.ResolveCouchRemote(namespace)
	if err != nil {
		t.Fatal(err)
	}
	origins, err := remotesession.OpenOriginStore(paths.Origins())
	if err != nil {
		t.Fatal(err)
	}
	return &messageService{origins: origins, originNamespace: namespace}
}

func serveForgetOwner(t *testing.T, namespace string) *messageService {
	t.Helper()
	owner := newForgetOwner(t, namespace)
	socket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		t.Fatal(err)
	}
	server, err := couchmessage.StartServer(t.Context(), socket, func(_ context.Context, raw []byte) ([]byte, error) {
		var request couchmessage.Request
		if err := json.Unmarshal(raw, &request); err != nil {
			return nil, err
		}
		return json.Marshal(owner.handleSessionForget(request))
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Error(err)
		}
	})
	return owner
}

func TestSessionSyncForgetUsesOwnerAndRetiresOnlyExactIntent(t *testing.T) {
	rt, file := sessionSyncRT(t)
	owner := serveForgetOwner(t, rt.namespace.Dir())
	syncer := sessionsync.New(rt.namespace.Dir())
	registration, err := syncer.Register(file, "ops:0")
	if err != nil {
		t.Fatal(err)
	}
	id := registration.Registration.ID
	if err := owner.origins.MarkRemote(id); err != nil {
		t.Fatal(err)
	}
	const other = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := owner.origins.MarkRemote(other); err != nil {
		t.Fatal(err)
	}
	code, out, errout := runSS(t, rt, "forget", file, "--json")
	if code != 0 || !strings.Contains(out, `"Changed":true`) {
		t.Fatalf("forget: %d %s %s", code, out, errout)
	}
	if got, err := syncer.RegistrationForPath(file); err != nil || got != nil {
		t.Fatalf("still registered: %+v %v", got, err)
	}
	if marked, err := owner.origins.Marked(id); err != nil || marked {
		t.Fatal("forgotten origin retained", err)
	}
	if marked, err := owner.origins.Marked(other); err != nil || !marked {
		t.Fatal("unrequested absent ID retired", err)
	}
	if code, out, errout = runSS(t, rt, "forget", file, "--json"); code != 0 || !strings.Contains(out, `"Changed":false`) {
		t.Fatalf("retry: %d %s %s", code, out, errout)
	}
}

func TestSessionSyncForgetReconcilesOnlyAbsentPersistedIntent(t *testing.T) {
	rt, file := sessionSyncRT(t)
	owner := newForgetOwner(t, rt.namespace.Dir())
	syncer := sessionsync.New(rt.namespace.Dir())
	registration, err := syncer.Register(file, "ops:0")
	if err != nil {
		t.Fatal(err)
	}
	id := registration.Registration.ID
	const absent = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const unrequested = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	for _, id := range []string{id, absent, unrequested} {
		if err := owner.origins.MarkRemote(id); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{id, absent} {
		if err := owner.origins.BeginForget(id); err != nil {
			t.Fatal(err)
		}
	}
	owner = newForgetOwner(t, rt.namespace.Dir())
	if response := owner.handleSessionForget(couchmessage.Request{Op: "session-forget", Target: file + ".absent"}); response.Code != "ok" {
		t.Fatal(response)
	}
	if marked, err := owner.origins.Marked(absent); err != nil || marked {
		t.Fatal("absent intent not retired", err)
	}
	for _, id := range []string{id, unrequested} {
		if marked, err := owner.origins.Marked(id); err != nil || !marked {
			t.Fatal("present/unrequested ID retired", id, err)
		}
	}
}

func TestSessionSyncForgetKeepsSubmittedProvenance(t *testing.T) {
	rt, file := sessionSyncRT(t)
	owner := newForgetOwner(t, rt.namespace.Dir())
	syncer := sessionsync.New(rt.namespace.Dir())
	registration, err := syncer.Register(file, "ops:0")
	if err != nil {
		t.Fatal(err)
	}
	id := registration.Registration.ID
	if err := owner.origins.MarkRemote(id); err != nil {
		t.Fatal(err)
	}
	if err := owner.origins.MarkSubmitted(id); err != nil {
		t.Fatal(err)
	}
	ledger := &sessionsync.Ledger{Dir: sessionsync.LedgerDir(rt.namespace.Dir())}
	if err := ledger.Append("ops:0", sessionsync.Entry{ID: "uncertain", From: "pair:6", At: time.Now(), Head: "submitted: " + file, DocumentID: id, RemoteEdited: true, Uncertain: true}); err != nil {
		t.Fatal(err)
	}
	response := owner.handleSessionForget(couchmessage.Request{Op: "session-forget", Target: file})
	if response.Code == "ok" {
		t.Fatal("uncertain receipt forgotten")
	}
	if got, err := syncer.RegistrationForPath(file); err != nil || got == nil || got.ID != id {
		t.Fatalf("uncertain registration removed: %+v %v", got, err)
	}
	if marked, err := owner.origins.Marked(id); err != nil || !marked {
		t.Fatal("uncertain provenance retired", err)
	}
}

func TestSessionSyncForgetRefusesInvalidLocalRequest(t *testing.T) {
	rt, file := sessionSyncRT(t)
	owner := newForgetOwner(t, rt.namespace.Dir())
	syncer := sessionsync.New(rt.namespace.Dir())
	first, err := syncer.Register(file, "ops:0")
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []couchmessage.Request{
		{Op: "session-forget", Target: file, Body: "delete"},
		{Op: "session-forget", Target: file, Scope: "forged"},
		{Op: "session-forget", Target: "relative.md"},
		{Op: "remote-off"},
	} {
		if response := owner.handleSessionForget(request); response.Code != "invalid-request" {
			t.Fatalf("request accepted: %+v -> %+v", request, response)
		}
	}
	got, err := syncer.RegistrationForPath(file)
	if err != nil || got == nil || got.ID != first.Registration.ID {
		t.Fatalf("invalid request changed registration: %+v %v", got, err)
	}
}

func TestSessionSyncForgetSubmitSeenNeverExpires(t *testing.T) {
	for _, marked := range []bool{false, true} {
		t.Run(fmt.Sprint("marked=", marked), func(t *testing.T) {
			rt, file := sessionSyncRT(t)
			owner := newForgetOwner(t, rt.namespace.Dir())
			syncer := sessionsync.New(rt.namespace.Dir())
			registration, err := syncer.Register(file, "ops:0")
			if err != nil {
				t.Fatal(err)
			}
			id := registration.Registration.ID
			if marked {
				if err := owner.origins.MarkRemote(id); err != nil {
					t.Fatal(err)
				}
			}
			if err := owner.origins.MarkSubmitted(id); err != nil {
				t.Fatal(err)
			}
			paths, err := artifactpath.ResolveCouchRemote(rt.namespace.Dir())
			if err != nil {
				t.Fatal(err)
			}
			old := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
			if err := os.Chtimes(paths.Origins(), old, old); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(paths.Audit(), []byte("invalid audit must not be read"), 0600); err != nil {
				t.Fatal(err)
			}
			owner = newForgetOwner(t, rt.namespace.Dir())
			response := owner.handleSessionForget(couchmessage.Request{Op: "session-forget", Target: file})
			if response.Code != "refused" || !strings.Contains(response.Error, remoteForgetFollowup) {
				t.Fatalf("aged submit forgotten: %+v", response)
			}
			if got, err := syncer.RegistrationForPath(file); err != nil || got == nil || got.ID != id {
				t.Fatalf("registration removed: %+v %v", got, err)
			}
		})
	}
}
