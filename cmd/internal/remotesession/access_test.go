package remotesession

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

const fixtureIssuer = "https://operator.cloudflareaccess.com"

type accessIssuerFixture struct {
	key    *rsa.PrivateKey
	keys   jose.JSONWebKeySet
	mu     sync.Mutex
	count  atomic.Int32
	server *httptest.Server
}

func newAccessIssuerFixture(t *testing.T) *accessIssuerFixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &accessIssuerFixture{key: key, keys: jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "first", Algorithm: "RS256", Use: "sig"}}}}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.count.Add(1)
		if r.URL.Path != "/cdn-cgi/access/certs" {
			t.Errorf("untrusted key path %s", r.URL.Path)
			w.WriteHeader(404)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(f.keys)
	}))
	t.Cleanup(f.server.Close)
	return f
}

type accessFixtureTransport struct {
	target string
	base   http.RoundTripper
}

func (r accessFixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	copy := req.Clone(req.Context())
	target := strings.TrimPrefix(r.target, "http://")
	copy.URL.Scheme = "http"
	copy.URL.Host = target
	return r.base.RoundTrip(copy)
}
func (f *accessIssuerFixture) verifier(t *testing.T, now func() time.Time) *AccessVerifier {
	t.Helper()
	v, err := newAccessVerifier(AccessConfig{Issuer: fixtureIssuer, Audience: "app", AllowedEmails: []string{"human@example.com"}}, accessFixtureTransport{f.server.URL, http.DefaultTransport}, now)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func (f *accessIssuerFixture) token(t *testing.T, mutate func(map[string]any), kid string) string {
	t.Helper()
	claims := map[string]any{"iss": fixtureIssuer, "aud": []string{"app"}, "sub": "user-id", "email": "human@example.com", "type": "app", "exp": time.Now().Add(time.Hour).Unix(), "nbf": time.Now().Add(-time.Minute).Unix()}
	if mutate != nil {
		mutate(claims)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: f.key}, (&jose.SignerOptions{}).WithHeader("kid", kid))
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	obj, err := signer.Sign(data)
	if err != nil {
		t.Fatal(err)
	}
	token, err := obj.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	return token
}
func TestAccessVerifierSignedIdentity(t *testing.T) {
	f := newAccessIssuerFixture(t)
	v := f.verifier(t, time.Now)
	token := f.token(t, nil, "first")
	id, err := v.Verify(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if id.email != "human@example.com" || id.subject != "user-id" || !id.validAt(time.Now()) {
		t.Fatalf("invalid verified identity: %+v", id)
	}
	for _, n := range []int{8, 16} {
		var wg sync.WaitGroup
		for range n {
			wg.Go(func() {
				if _, err := v.Verify(context.Background(), token); err != nil {
					t.Error(err)
				}
			})
		}
		wg.Wait()
	}
	if got := f.count.Load(); got != 1 {
		t.Fatalf("key fetches=%d, want1", got)
	}
}
func TestAccessVerifierRejectsClaims(t *testing.T) {
	f := newAccessIssuerFixture(t)
	v := f.verifier(t, time.Now)
	tests := map[string]func(map[string]any){"issuer": func(c map[string]any) { c["iss"] = "https://evil.example" }, "audience": func(c map[string]any) { c["aud"] = []string{"other"} }, "expiry": func(c map[string]any) { c["exp"] = time.Now().Add(-time.Second).Unix() }, "not-before": func(c map[string]any) { c["nbf"] = time.Now().Add(time.Minute).Unix() }, "no-subject": func(c map[string]any) { delete(c, "sub") }, "no-email": func(c map[string]any) { delete(c, "email") }, "service": func(c map[string]any) { c["type"] = "service" }, "unallowed": func(c map[string]any) { c["email"] = "other@example.com" }, "missing-expiry": func(c map[string]any) { delete(c, "exp") }}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			token := f.token(t, mutate, "first")
			if _, err := v.Verify(context.Background(), token); err == nil {
				t.Fatal("accepted invalid claims")
			} else if strings.Contains(err.Error(), token) {
				t.Fatal("token leaked")
			}
		})
	}
	for _, token := range []string{"", strings.Repeat("x", 8193), "a.b.c"} {
		if _, err := v.Verify(context.Background(), token); err == nil {
			t.Fatal("accepted malformed assertion")
		}
	}
}
func TestAccessConfigPinned(t *testing.T) {
	for _, issuer := range []string{"http://operator.cloudflareaccess.com", "https://evil.example", "https://a.b.cloudflareaccess.com", "https://operator.cloudflareaccess.com/", "https://operator.cloudflareaccess.com:443", "https://x@operator.cloudflareaccess.com", "https://operator.cloudflareaccess.com?x=1"} {
		if _, err := NewAccessVerifier(AccessConfig{Issuer: issuer, Audience: "app", AllowedEmails: []string{"human@example.com"}}); err == nil {
			t.Errorf("accepted %s", issuer)
		}
	}
}
