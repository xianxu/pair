package broadcast

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNoteOwnerFollowsValidatedSession(t *testing.T) {
	s := startSession(t, &FakeTunnel{}, HubOptions{})
	a, b := jarClient(t), jarClient(t)
	code := codeOf(t, s, LinkView)
	for _, c := range []*http.Client{a, b} {
		if status, _, _ := unlockWith(t, c, s.Link(), code); status != http.StatusOK {
			t.Fatalf("unlock: %d", status)
		}
	}
	owner := func(c *http.Client) string {
		t.Helper()
		status, _, h := do(t, c, http.MethodGet, s.Link()+"session", "", "", nil)
		if status != http.StatusNoContent {
			t.Fatalf("session: %d", status)
		}
		return h.Get("X-Couch-Note-Owner")
	}
	first := owner(a)
	if len(first) != 64 || first == owner(b) {
		t.Fatalf("missing or shared owner: %q", first)
	}
	if owner(a) != first {
		t.Fatal("refresh changed owner")
	}
	if status, _, _ := unlockWith(t, a, s.Link(), code); status != http.StatusOK {
		t.Fatalf("re-unlock: %d", status)
	}
	if owner(a) == first {
		t.Fatal("new unlock retained old ownership")
	}
	if err := s.RotateCode(LinkView); err != nil {
		t.Fatal(err)
	}
	status, _, h := do(t, a, http.MethodGet, s.Link()+"session", "", "", nil)
	if status != http.StatusUnauthorized || h.Get("X-Couch-Note-Owner") != "" {
		t.Fatal("revoked cookie projected owner")
	}
}

func TestNoteOwnerCannotBeClaimedByRequest(t *testing.T) {
	l, err := NewLinkLock(4, nil)
	if err != nil {
		t.Fatal(err)
	}
	session, _ := l.Unlock(time.Now(), l.Code())
	r := httptest.NewRequest(http.MethodGet, "/link/events?owner=operator", nil)
	r.Header.Set(NoteOwnerHeader, OperatorNoteOwner)
	if NoteOwnerFromContext(r.Context()) != "" || NoteOwnerFromContext(context.Background()) != "" {
		t.Fatal("untrusted request has owner")
	}
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: session})
	w := httptest.NewRecorder()
	accepted, done := gate(w, r, "link", "events", l, false)
	if accepted == nil {
		t.Fatal("valid cookie refused")
	}
	defer done()
	owner := NoteOwnerFromContext(accepted.Context())
	if owner == "" || owner == OperatorNoteOwner || owner == session || strings.Contains(owner, session) {
		t.Fatal("owner is forged or exposes credential")
	}
	if w.Header().Get(NoteOwnerHeader) != "" {
		t.Fatal("unneeded owner header on stream")
	}
}

func TestNoteOwnerRevocationIsSynchronous(t *testing.T) {
	revoked := make(chan struct{})
	// A live parent context deliberately has no cancellation goroutine.
	ctx := context.WithValue(context.Background(), noteOwnerKey{}, noteOwnerProof{owner: "owner", revoked: revoked})
	if NoteOwnerFromContext(ctx) != "owner" {
		t.Fatal("live proof missing")
	}
	close(revoked)
	if NoteOwnerFromContext(ctx) != "" {
		t.Fatal("revoked proof retained authority")
	}
}
