package remotesession

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

type documentHTTPFixture struct {
	server      *RemoteServer
	supervisor  *Supervisor
	documents   *fixtureDocuments
	token, path string
	headers     map[string]string
}

func newDocumentHTTPFixture(t *testing.T) documentHTTPFixture {
	t.Helper()
	issuer := newAccessIssuerFixture(t)
	s, _ := supervisorFixture(t, &broadcast.FakeTunnel{})
	f := newFixtureDocuments(t, DocumentSnapshot{Ref: DocumentRef{ID: originDocA, Generation: "editor-1"}, Revision: "r1", Text: "hello"})
	s.backend = f
	server := newRemoteServer(s, issuer.verifier(t, time.Now))
	s.handler = server
	if err := s.On(); err != nil {
		t.Fatal(err)
	}
	awaitRemote(t, s, "on")
	token := issuer.token(t, nil, "first")
	path := strings.TrimPrefix(s.ConsoleLink(), "https://remote.example.com")
	w := remoteRequest(t, server, "POST", path+"handshake", token, "{}", map[string]string{"X-Remote-Handshake": "1"})
	if w.Code != 200 {
		t.Fatalf("handshake=%d %s", w.Code, w.Body.String())
	}
	var result struct{ CSRF string }
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return documentHTTPFixture{server, s, f, token, path, map[string]string{"Cookie": w.Result().Cookies()[0].String(), "X-Remote-CSRF": result.CSRF}}
}
func (f documentHTTPFixture) request(t *testing.T, route string, request DocumentRequest) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	return remoteRequest(t, f.server, "POST", f.path+"v1/"+route, f.token, string(raw), f.headers)
}
func TestRemoteDocumentRoutesRequireSessionAndRejectUnsafeRequests(t *testing.T) {
	f := newDocumentHTTPFixture(t)
	if w := remoteRequest(t, f.server, "GET", f.path+"v1/documents", f.token, "", nil); w.Code != 401 {
		t.Fatalf("sessionless list=%d", w.Code)
	}
	if w := remoteRequest(t, f.server, "GET", f.path+"v1/documents", f.token, "", f.headers); w.Code != 200 || !strings.Contains(w.Body.String(), originDocA) {
		t.Fatalf("list=%d %s", w.Code, w.Body.String())
	}
	read := DocumentRequest{Sequence: 1, Action: ActionRead, Document: DocumentRef{ID: originDocA, Generation: "editor-1"}}
	if w := f.request(t, "read", read); w.Code != 200 || !strings.Contains(w.Body.String(), "hello") {
		t.Fatalf("read=%d %s", w.Code, w.Body.String())
	}
	bad := map[string]string{"Cookie": f.headers["Cookie"]}
	if w := remoteRequest(t, f.server, "POST", f.path+"v1/read", f.token, `{"sequence":1,"action":"read","document":{"id":"`+originDocA+`","generation":"editor-1"}}`, bad); w.Code != 401 {
		t.Fatalf("CSRF-less read=%d", w.Code)
	}
	for _, body := range []string{
		`{"sequence":1,"action":"read","document":{"id":"` + originDocA + `","generation":"editor-1"},"Origin":"remote"}`,
		`{"sequence":1,"action":"read","document":{"id":"/etc/passwd","generation":"editor-1"}}`,
		`{"sequence":1,"action":"submit","document":{"id":"` + originDocA + `","generation":"editor-1"},"base_revision":"r1"}`,
	} {
		if w := remoteRequest(t, f.server, "POST", f.path+"v1/read", f.token, body, f.headers); w.Code != 400 {
			t.Fatalf("unsafe body accepted=%d %s", w.Code, w.Body.String())
		}
	}
	read.Document.Generation = "stale"
	if w := f.request(t, "read", read); w.Code != 409 {
		t.Fatalf("stale read=%d %s", w.Code, w.Body.String())
	}
	for _, route := range []string{"files", "exec", "rpc", "reset", "on"} {
		if w := f.request(t, route, read); w.Code != 404 {
			t.Fatalf("unexpected generic route %s=%d", route, w.Code)
		}
	}
	f.supervisor.backend = nil
	if w := remoteRequest(t, f.server, "GET", f.path+"v1/documents", f.token, "", f.headers); w.Code != 503 {
		t.Fatalf("missing backend=%d", w.Code)
	}
}
func TestRemoteDocumentHTTPDispatchCountsUniqueAndDuplicateAtNAnd2N(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			f := newDocumentHTTPFixture(t)
			counted := &countingHTTPDocuments{fixtureDocuments: f.documents, sources: make(chan DocumentSource, n)}
			f.supervisor.backend = counted
			for i := 1; i <= n; i++ {
				for range 2 {
					w := f.request(t, "submit", submitRequest(uint64(i)))
					if w.Code != 200 || !strings.Contains(w.Body.String(), "applied") {
						t.Fatalf("dispatch=%d %s", w.Code, w.Body.String())
					}
				}
			}
			f.documents.mu.Lock()
			effects := f.documents.effects
			f.documents.mu.Unlock()
			if counted.reads.Load() != int64(n) {
				t.Fatalf("backend reads=%d want%d", counted.reads.Load(), n)
			}
			for range n {
				source := <-counted.sources
				if source != (DocumentSource{Email: "human@example.com", IP: "192.0.2.9"}) {
					t.Fatalf("unverified provenance: %+v", source)
				}
			}
			if effects != n {
				t.Fatalf("effects=%d want%d", effects, n)
			}
			_, audit, err := readAuditSegment(f.supervisor.paths.Audit())
			if err != nil || len(audit) != 2*n {
				t.Fatalf("audit=%d err=%v", len(audit), err)
			}
		})
	}
}

// The stream fixture extends the same stateful backend. It owns no producer
// goroutine; the handler owns reading and canceling this subscription.
type httpStreamDocuments struct {
	*fixtureDocuments
	snapshots chan DocumentSnapshot
	entered   chan context.Context
}

func (f *httpStreamDocuments) Subscribe(ctx context.Context, ref DocumentRef) (<-chan DocumentSnapshot, error) {
	if _, err := f.Read(ctx, ref); err != nil {
		return nil, err
	}
	f.entered <- ctx
	return f.snapshots, nil
}
func TestRemoteDocumentStreamOnePerSessionAndKillJoins(t *testing.T) {
	f := newDocumentHTTPFixture(t)
	stream := &httpStreamDocuments{fixtureDocuments: f.documents, snapshots: make(chan DocumentSnapshot, 1), entered: make(chan context.Context, 1)}
	f.supervisor.backend = stream
	request := DocumentRequest{Sequence: 1, Action: ActionRead, Document: DocumentRef{ID: originDocA, Generation: "editor-1"}}
	raw, _ := json.Marshal(request)
	first := httptest.NewRequest("POST", "https://remote.example.com"+f.path+"v1/stream", strings.NewReader(string(raw)))
	first.Header.Set("Cf-Access-Jwt-Assertion", f.token)
	first.Header.Set("Cf-Connecting-IP", "192.0.2.9")
	first.Header.Set("Origin", "https://remote.example.com")
	first.Header.Set("Content-Type", "application/json")
	for key, value := range f.headers {
		first.Header.Set(key, value)
	}
	finished := make(chan struct{})
	go func() { defer close(finished); f.server.ServeHTTP(httptest.NewRecorder(), first) }()
	var subscription context.Context
	select {
	case subscription = <-stream.entered:
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
	// The admission's five-second deadline must not become the stream lifetime.
	deadline, ok := subscription.Deadline()
	if !ok || time.Until(deadline) <= 5*time.Second {
		t.Fatalf("stream inherited exchange deadline: %v", deadline)
	}
	if w := f.request(t, "stream", request); w.Code != 429 {
		t.Fatalf("second session stream=%d %s", w.Code, w.Body.String())
	}
	f.supervisor.Kill()
	select {
	case <-subscription.Done():
	case <-t.Context().Done():
		t.Fatal("subscription not revoked")
	}
	select {
	case <-finished:
	case <-t.Context().Done():
		t.Fatal("stream handler not joined")
	}
}

type countingHTTPDocuments struct {
	*fixtureDocuments
	reads   atomic.Int64
	sources chan DocumentSource
}

func (f *countingHTTPDocuments) Read(ctx context.Context, ref DocumentRef) (DocumentSnapshot, error) {
	f.reads.Add(1)
	return f.fixtureDocuments.Read(ctx, ref)
}
func (f *countingHTTPDocuments) Dispatch(ctx context.Context, request DocumentRequest, permit Permit) (<-chan DocumentResult, bool) {
	select {
	case f.sources <- permit.Source():
	default:
	}
	return f.fixtureDocuments.Dispatch(ctx, request, permit)
}

func TestRemoteDocumentStreamRefusesOversizeFrameAndReleasesSession(t *testing.T) {
	f := newDocumentHTTPFixture(t)
	stream := &httpStreamDocuments{fixtureDocuments: f.documents, snapshots: make(chan DocumentSnapshot, 1), entered: make(chan context.Context, 1)}
	f.supervisor.backend = stream
	request := DocumentRequest{Sequence: 1, Action: ActionRead, Document: DocumentRef{ID: originDocA, Generation: "editor-1"}}
	finished := make(chan *httptest.ResponseRecorder, 1)
	go func() { finished <- f.request(t, "stream", request) }()
	subscription := <-stream.entered
	stream.snapshots <- DocumentSnapshot{Ref: request.Document, Revision: "r1", Text: strings.Repeat("x", remoteDocumentFrameBytes)}
	select {
	case response := <-finished:
		if response.Body.Len() != 0 {
			t.Fatal("oversized frame leaked output")
		}
	case <-t.Context().Done():
		t.Fatal("oversized stream not closed")
	}
	if subscription.Err() == nil {
		t.Fatal("subscription context left running")
	}
	// The slot was released by the failed stream; another subscription can open.
	go func() { finished <- f.request(t, "stream", request) }()
	select {
	case <-stream.entered:
	case <-t.Context().Done():
		t.Fatal("stream slot leaked")
	}
	f.supervisor.Kill()
	select {
	case <-finished:
	case <-t.Context().Done():
		t.Fatal("replacement stream not joined")
	}
}

func TestRemoteDocumentFrameBoundIncludesJSONAndNewline(t *testing.T) {
	snapshot := DocumentSnapshot{Ref: DocumentRef{ID: originDocA, Generation: "editor-1"}, Revision: "r1"}
	empty, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Text = strings.Repeat("x", remoteDocumentFrameBytes-len(empty)-1)
	if raw, err := boundedDocumentJSON(snapshot); err != nil || len(raw)+1 != remoteDocumentFrameBytes {
		t.Fatalf("exact bound size=%d err=%v", len(raw)+1, err)
	}
	snapshot.Text += "x"
	if _, err := boundedDocumentJSON(snapshot); err == nil {
		t.Fatal("frame bound exceeded")
	}
}

func TestRemoteDocumentEditAndNavigationUseRegisteredState(t *testing.T) {
	f := newDocumentHTTPFixture(t)
	edit := DocumentRequest{Sequence: 1, Action: ActionEdit, Document: DocumentRef{ID: originDocA, Generation: "editor-1"}, BaseRevision: "r1", Edit: &DocumentEdit{Start: 0, End: 5, Text: "changed"}}
	if w := f.request(t, "edit", edit); w.Code != 200 {
		t.Fatalf("edit=%d %s", w.Code, w.Body.String())
	}
	f.documents.mu.Lock()
	snapshot := f.documents.documents[originDocA]
	destination := DocumentRef{ID: originDocB, Generation: "editor-2"}
	f.documents.documents[originDocB] = DocumentSnapshot{Ref: destination, Revision: "target", Text: "side doc"}
	f.documents.links["link:allowed"] = destination
	f.documents.mu.Unlock()
	if snapshot.Text != "changed" {
		t.Fatal("edit not applied to stateful document")
	}
	navigation := DocumentRequest{Sequence: 2, Action: ActionNavigate, Document: edit.Document, BaseRevision: snapshot.Revision, Navigation: &NavigationTarget{Kind: "link", Key: "allowed"}}
	if w := f.request(t, "navigate", navigation); w.Code != 200 {
		t.Fatalf("navigation=%d %s", w.Code, w.Body.String())
	}
	f.documents.mu.Lock()
	defer f.documents.mu.Unlock()
	if len(f.documents.opened) != 1 || f.documents.opened[0] != destination {
		t.Fatalf("opened=%+v", f.documents.opened)
	}
}

func TestRemoteDocumentHTTPRejectsDuplicateAndOversizeBody(t *testing.T) {
	f := newDocumentHTTPFixture(t)
	for _, body := range []string{
		`{"sequence":1,"sequence":2,"action":"read","document":{"id":"` + originDocA + `","generation":"editor-1"}}`,
		`{"sequence":1,"action":"read","document":{"id":"` + originDocA + `","id":"` + originDocB + `","generation":"editor-1"}}`,
		strings.Repeat(" ", (64<<10)+1) + `{}`,
	} {
		if w := remoteRequest(t, f.server, "POST", f.path+"v1/read", f.token, body, f.headers); w.Code != 400 {
			t.Fatalf("unsafe body=%d %s", w.Code, w.Body.String())
		}
	}
}

func TestRemoteDocumentStreamLogoutCancelsSubscription(t *testing.T) {
	f := newDocumentHTTPFixture(t)
	stream := &httpStreamDocuments{fixtureDocuments: f.documents, snapshots: make(chan DocumentSnapshot, 1), entered: make(chan context.Context, 1)}
	f.supervisor.backend = stream
	request := DocumentRequest{Sequence: 1, Action: ActionRead, Document: DocumentRef{ID: originDocA, Generation: "editor-1"}}
	finished := make(chan struct{})
	go func() { defer close(finished); f.request(t, "stream", request) }()
	subscription := <-stream.entered
	if w := remoteRequest(t, f.server, "POST", f.path+"logout", f.token, "{}", f.headers); w.Code != 204 {
		t.Fatalf("logout=%d", w.Code)
	}
	select {
	case <-finished:
	case <-t.Context().Done():
		t.Fatal("logout did not join stream")
	}
	if subscription.Err() == nil {
		t.Fatal("logout retained stream context")
	}
}

type heldHTTPReadDocuments struct {
	*fixtureDocuments
	entered chan struct{}
}

func (f *heldHTTPReadDocuments) Read(ctx context.Context, ref DocumentRef) (DocumentSnapshot, error) {
	f.entered <- struct{}{}
	<-ctx.Done()
	return DocumentSnapshot{}, ctx.Err()
}
func TestRemoteDocumentRequestLanesReserveKillCapacity(t *testing.T) {
	f := newDocumentHTTPFixture(t)
	held := &heldHTTPReadDocuments{fixtureDocuments: f.documents, entered: make(chan struct{}, 6)}
	f.supervisor.backend = held
	request := DocumentRequest{Sequence: 1, Action: ActionRead, Document: DocumentRef{ID: originDocA, Generation: "editor-1"}}
	finished := make(chan *httptest.ResponseRecorder, 6)
	for range 6 {
		go func() { finished <- f.request(t, "read", request) }()
	}
	for range 6 {
		select {
		case <-held.entered:
		case <-t.Context().Done():
			t.Fatal("read lane not entered")
		}
	}
	if w := f.request(t, "read", request); w.Code != 429 {
		t.Fatalf("seventh ordinary request=%d", w.Code)
	}
	if w := remoteRequest(t, f.server, "POST", "/remote/kill", f.token, "{}", map[string]string{"X-Remote-Kill": "1"}); w.Code != 204 {
		t.Fatalf("reserved kill=%d", w.Code)
	}
	for range 6 {
		select {
		case <-finished:
		case <-t.Context().Done():
			t.Fatal("read handler not joined after kill")
		}
	}
}

type receiptHTTPDocuments struct{ *fixtureDocuments }

func (f *receiptHTTPDocuments) Dispatch(ctx context.Context, request DocumentRequest, permit Permit) (<-chan DocumentResult, bool) {
	input, accepted := f.fixtureDocuments.Dispatch(ctx, request, permit)
	if !accepted {
		return nil, false
	}
	output := make(chan DocumentResult, 1)
	go func() {
		select {
		case result := <-input:
			result.ReceiptID = "receipt-1"
			output <- result
		case <-ctx.Done():
			output <- DocumentResult{Outcome: "uncertain"}
		}
	}()
	return output, true
}
func TestRemoteDocumentHTTPUncertainResultPreservesReceipt(t *testing.T) {
	f := newDocumentHTTPFixture(t)
	f.supervisor.backend = &receiptHTTPDocuments{f.documents}
	original := f.supervisor.audit.appendLine
	f.supervisor.audit.appendLine = func(path string, raw []byte) error {
		if strings.Contains(string(raw), `"kind":"result"`) {
			return errors.New("unavailable")
		}
		return original(path, raw)
	}
	response := f.request(t, "submit", submitRequest(1))
	var decoded struct {
		Result DocumentResult
		Error  string
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("non-JSON failure: %d %s %v", response.Code, response.Body.String(), err)
	}
	if response.Code != 409 || decoded.Result.Outcome != "uncertain" || decoded.Result.ReceiptID != "receipt-1" || decoded.Error == "" {
		t.Fatalf("uncertainty lost receipt: %d %+v", response.Code, decoded)
	}
}
