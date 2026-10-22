package remotesession

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

func remoteRequest(t *testing.T, handler http.Handler, method, path, token, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, "https://remote.example.com"+path, bytes.NewBufferString(body))
	r.Header.Set("Cf-Access-Jwt-Assertion", token)
	r.Header.Set("Cf-Connecting-IP", "192.0.2.9")
	if method == "POST" {
		r.Header.Set("Origin", "https://remote.example.com")
		r.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}
func TestRemoteServerSignedBoundaryAndSessionResume(t *testing.T) {
	issuer := newAccessIssuerFixture(t)
	s, _ := supervisorFixture(t, &broadcast.FakeTunnel{})
	server := newRemoteServer(s, issuer.verifier(t, time.Now))
	s.handler = server
	if err := s.On(); err != nil {
		t.Fatal(err)
	}
	awaitRemote(t, s, "on")
	path := strings.TrimPrefix(s.ConsoleLink(), "https://remote.example.com")
	for _, candidate := range []string{"", "forged"} {
		if w := remoteRequest(t, server, "GET", path, candidate, "", nil); w.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated page: %d", w.Code)
		}
	}
	token := issuer.token(t, nil, "first")
	if w := remoteRequest(t, server, "GET", path, token, "", nil); w.Code != 200 {
		t.Fatalf("page=%d %s", w.Code, w.Body.String())
	}
	h := map[string]string{"X-Remote-Handshake": "1"}
	w := remoteRequest(t, server, "POST", path+"handshake", token, "{}", h)
	if w.Code != 200 {
		t.Fatalf("handshake=%d %s", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteStrictMode || cookies[0].Path != path {
		t.Fatalf("unsafe cookie: %+v", cookies)
	}
	var result struct {
		CSRF    string
		Expires time.Time
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || len(result.CSRF) != 43 {
		t.Fatalf("bad handshake: %v %s", err, w.Body.String())
	}
	if strings.Contains(w.Body.String(), cookies[0].Value) {
		t.Fatal("HttpOnly credential leaked in JSON")
	}
	h["Cookie"] = cookies[0].String()
	h["X-Remote-CSRF"] = result.CSRF
	for range 8 {
		w = remoteRequest(t, server, "POST", path+"handshake", token, "{}", h)
		if w.Code != 200 {
			t.Fatalf("refresh allocated another session: %d %s", w.Code, w.Body.String())
		}
	}
	if w = remoteRequest(t, server, "POST", path+"logout", token, "{}", h); w.Code != 204 {
		t.Fatalf("logout=%d", w.Code)
	}
	if w = remoteRequest(t, server, "POST", path+"logout", token, "{}", h); w.Code != 401 {
		t.Fatalf("logged out cookie retained authority=%d", w.Code)
	}
}
func TestRemoteServerKillOutsidePasscodeAndSessionCap(t *testing.T) {
	issuer := newAccessIssuerFixture(t)
	s, _ := supervisorFixture(t, &broadcast.FakeTunnel{})
	verifier, err := newAccessVerifier(AccessConfig{
		Issuer: fixtureIssuer, Audience: "app",
		AllowedEmails: []string{"human@example.com", "second@example.com"},
	}, accessFixtureTransport{issuer.server.URL, http.DefaultTransport}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	server := newRemoteServer(s, verifier)
	s.handler = server
	if err := s.On(); err != nil {
		t.Fatal(err)
	}
	awaitRemote(t, s, "on")
	path := strings.TrimPrefix(s.ConsoleLink(), "https://remote.example.com")
	token := issuer.token(t, nil, "first")
	for range 4 {
		w := remoteRequest(t, server, "POST", path+"handshake", token, "{}", map[string]string{"X-Remote-Handshake": "1"})
		if w.Code != 200 {
			t.Fatal(w.Code)
		}
	}
	if w := remoteRequest(t, server, "POST", path+"handshake", token, "{}", map[string]string{"X-Remote-Handshake": "1"}); w.Code != 429 {
		t.Fatal("fifth session admitted", w.Code)
	}
	for _, headers := range []map[string]string{nil, {"Origin": "https://evil.example", "X-Remote-Kill": "1"}, {"X-Remote-Kill": "wrong"}} {
		if w := remoteRequest(t, server, "POST", "/remote/kill", token, "{}", headers); w.Code < 400 {
			t.Fatal("unsafe kill admitted", w.Code)
		}
	}
	// A separately authenticated allowed user needs neither the initiating
	// user's cookie nor a free session slot nor the per-enable link passcode.
	second := issuer.token(t, func(claims map[string]any) {
		claims["sub"] = "second-user-id"
		claims["email"] = "second@example.com"
	}, "first")
	if w := remoteRequest(t, server, "POST", "/remote/kill", second, "{}", map[string]string{"X-Remote-Kill": "1"}); w.Code != 204 {
		t.Fatalf("sessionless kill failed: %d %s", w.Code, w.Body.String())
	}
	if s.Snapshot().Status != "disabled" || s.ConsoleLink() != "" {
		t.Fatal("remote kill failed synchronous revocation")
	}
	if w := remoteRequest(t, server, "GET", path, token, "", nil); w.Code < 400 {
		t.Fatal("revoked capability still served")
	}
}
func TestRemoteServerNoGenericRoutes(t *testing.T) {
	issuer := newAccessIssuerFixture(t)
	s, _ := supervisorFixture(t, &broadcast.FakeTunnel{})
	server := newRemoteServer(s, issuer.verifier(t, time.Now))
	s.handler = server
	if err := s.On(); err != nil {
		t.Fatal(err)
	}
	awaitRemote(t, s, "on")
	token := issuer.token(t, nil, "first")
	path := strings.TrimPrefix(s.ConsoleLink(), "https://remote.example.com")
	for _, suffix := range []string{"drive", "files", "exec", "rpc", "reset", "on", "../kill", "%2e%2e/kill"} {
		for _, method := range []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"} {
			if w := remoteRequest(t, server, method, path+suffix, token, "{}", nil); w.Code < 400 {
				t.Fatalf("generic route admitted: %s %s %d", method, suffix, w.Code)
			}
		}
	}
}

func TestRemoteServerInvalidAssertionRevokesPresentedSession(t *testing.T) {
	issuer := newAccessIssuerFixture(t)
	s, _ := supervisorFixture(t, &broadcast.FakeTunnel{})
	server := newRemoteServer(s, issuer.verifier(t, time.Now))
	s.handler = server
	if err := s.On(); err != nil {
		t.Fatal(err)
	}
	awaitRemote(t, s, "on")
	path := strings.TrimPrefix(s.ConsoleLink(), "https://remote.example.com")
	token := issuer.token(t, nil, "first")
	w := remoteRequest(t, server, "POST", path+"handshake", token, "{}", map[string]string{"X-Remote-Handshake": "1"})
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	cookie := w.Result().Cookies()[0]
	var result struct{ CSRF string }
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	headers := map[string]string{"Cookie": cookie.String(), "X-Remote-CSRF": result.CSRF}
	w = remoteRequest(t, server, "GET", path+"status", "invalid", "", headers)
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
	id, err := server.verifier.Verify(t.Context(), token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.epoch.sessions.Admit(cookie.Value, result.CSRF, id, time.Now(), true); err == nil {
		t.Fatal("rejected Access assertion retained session")
	}
}

func TestRemoteServerThroughProtectedUnixOrigin(t *testing.T) {
	issuer := newAccessIssuerFixture(t)
	tunnel := &broadcast.FakeTunnel{}
	s, _ := supervisorFixture(t, tunnel)
	s.handler = newRemoteServer(s, issuer.verifier(t, time.Now))
	if err := s.On(); err != nil {
		t.Fatal(err)
	}
	awaitRemote(t, s, "on")
	address := tunnel.LastAddr()
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", address)
	}}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport}
	token := issuer.token(t, nil, "first")
	for _, n := range []int{8, 16} {
		before := issuer.count.Load()
		for range n {
			r, err := http.NewRequestWithContext(t.Context(), "GET", s.ConsoleLink(), nil)
			if err != nil {
				t.Fatal(err)
			}
			r.Header.Set("Cf-Access-Jwt-Assertion", token)
			r.Header.Set("Cf-Connecting-IP", "192.0.2.9")
			// Private origin is plain HTTP on Unix; Host and public-origin policy stay
			// the actual configured HTTPS hostname, exactly as cloudflared forwards it.
			r.URL.Scheme = "http"
			response, err := client.Do(r)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := io.ReadAll(response.Body)
			response.Body.Close()
			if err != nil || response.StatusCode != 200 || !bytes.Contains(raw, []byte("Pair remote")) {
				t.Fatalf("status=%d error=%v", response.StatusCode, err)
			}
		}
		// One initial bounded key fetch, never one fetch per HTTP request.
		if got := issuer.count.Load() - before; got > 1 {
			t.Fatalf("N=%d JWKS fetches=%d", n, got)
		}
	}
}
