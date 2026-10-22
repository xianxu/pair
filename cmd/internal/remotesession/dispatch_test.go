package remotesession

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

func dispatchFixture(t *testing.T) (*Supervisor, *fixtureDocuments, *remoteEpoch, sessionRef, AccessIdentity) {
	t.Helper()
	s, _ := supervisorFixture(t, &broadcast.FakeTunnel{})
	ref := DocumentRef{ID: originDocA, Generation: "editor-1"}
	f := newFixtureDocuments(t, DocumentSnapshot{Ref: ref, Revision: "r1", Text: "hello"})
	s.backend = f
	if err := s.On(); err != nil {
		t.Fatal(err)
	}
	awaitRemote(t, s, "on")
	s.gate.mu.Lock()
	e := s.epoch
	s.gate.mu.Unlock()
	id := AccessIdentity{issuer: "issuer", subject: "human", audience: "app", email: "human@example.com", expires: time.Now().Add(time.Hour)}
	creds, err := e.sessions.Handshake(id, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	session, err := e.sessions.Admit(creds.Cookie, creds.CSRF, id, time.Now(), true)
	if err != nil {
		t.Fatal(err)
	}
	return s, f, e, session, id
}
func submitRequest(sequence uint64) DocumentRequest {
	return DocumentRequest{Sequence: sequence, Action: ActionSubmit, Document: DocumentRef{ID: originDocA, Generation: "editor-1"}, BaseRevision: "r1"}
}
func TestRemoteDispatchUniqueAndDuplicateAtNAnd2N(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			s, f, e, session, id := dispatchFixture(t)
			for i := 1; i <= n; i++ {
				req := submitRequest(uint64(i))
				first, err := s.Dispatch(t.Context(), e, session, id, "192.0.2.9", req)
				if err != nil || first.Outcome != "applied" {
					t.Fatal(first, err)
				}
				again, err := s.Dispatch(t.Context(), e, session, id, "192.0.2.9", req)
				if err != nil || again != first {
					t.Fatal("duplicate", again, err)
				}
			}
			f.mu.Lock()
			effects := f.effects
			f.mu.Unlock()
			if effects != n {
				t.Fatalf("effects=%d want%d", effects, n)
			}
			_, rows, err := readAuditSegment(s.paths.Audit())
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 2*n {
				t.Fatalf("audit=%d want%d", len(rows), 2*n)
			}
			for _, row := range rows {
				raw, _ := json.Marshal(row)
				if strings.Contains(string(raw), id.email) {
					t.Fatal("raw identity in audit")
				}
			}
		})
	}
}
func TestRemoteDispatchIntentFailureHasNoEffect(t *testing.T) {
	s, f, e, session, id := dispatchFixture(t)
	s.audit.appendLine = func(string, []byte) error { return errors.New("disk full") }
	result, err := s.Dispatch(t.Context(), e, session, id, "192.0.2.9", submitRequest(1))
	if err == nil || result.Outcome == "applied" {
		t.Fatal("intent failed but admitted", result, err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.effects != 0 {
		t.Fatal("effect despite failed intent")
	}
}
func TestRemoteDispatchKillWhileIntentHeldHasNoEffect(t *testing.T) {
	s, f, e, session, id := dispatchFixture(t)
	entered, release := make(chan struct{}), make(chan struct{})
	original := s.audit.appendLine
	s.audit.appendLine = func(path string, raw []byte) error {
		if strings.Contains(string(raw), `"kind":"intent"`) {
			close(entered)
			<-release
		}
		return original(path, raw)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = s.Dispatch(context.Background(), e, session, id, "192.0.2.9", submitRequest(1))
	}()
	<-entered
	s.Kill() // Must not wait on the audit writer.
	close(release)
	<-done
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.effects != 0 {
		t.Fatal("effect crossed kill")
	}
}

func TestRemoteDispatchForgetFencesOnlyExactID(t *testing.T) {
	s, _, e, session, id := dispatchFixture(t)
	if err := s.FenceDocument(originDocA); err != nil {
		t.Fatal(err)
	}
	if result, err := s.Dispatch(t.Context(), e, session, id, "192.0.2.9", submitRequest(1)); err == nil || result.Outcome == "applied" {
		t.Fatal("forgotten ID admitted", result, err)
	}
}

func TestRemoteDispatchCanceledObserverBeforeAdmissionHasNoEffect(t *testing.T) {
	s, f, e, session, id := dispatchFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if result, err := s.Dispatch(ctx, e, session, id, "192.0.2.9", submitRequest(1)); err == nil || result.Outcome == "applied" {
		t.Fatal("canceled observer admitted", result, err)
	}
	e.requests.Wait()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.effects != 0 {
		t.Fatal("canceled request applied")
	}
}
func TestRemoteForgetFenceCompletionRequiresRetiredEvidence(t *testing.T) {
	s, _, _, _, _ := dispatchFixture(t)
	if err := s.origins.MarkRemote(originDocA); err != nil {
		t.Fatal(err)
	}
	if err := s.FenceDocument(originDocA); err != nil {
		t.Fatal(err)
	}
	if err := s.CompleteForget(originDocA); err == nil {
		t.Fatal("released live origin fence")
	}
	if err := s.origins.BeginForget(originDocA); err != nil {
		t.Fatal(err)
	}
	if err := s.origins.RetireForgotten(originDocA); err != nil {
		t.Fatal(err)
	}
	if err := s.CompleteForget(originDocA); err != nil {
		t.Fatal(err)
	}
	s.gate.mu.Lock()
	retained := s.gate.forgotten[originDocA]
	s.gate.mu.Unlock()
	if retained {
		t.Fatal("retired fence leaked")
	}
}
