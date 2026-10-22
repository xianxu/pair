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

func TestAccessKeysRotationAndUnknownWaves(t *testing.T) {
	f := newAccessIssuerFixture(t)
	clock := time.Now()
	v := f.verifier(t, func() time.Time { return clock })
	first := f.token(t, nil, "first")
	if _, err := v.Verify(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	secondKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.key = secondKey
	f.keys = jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &secondKey.PublicKey, KeyID: "second", Algorithm: "RS256", Use: "sig"}}}
	f.mu.Unlock()
	clock = clock.Add(accessRefreshInterval)
	second := f.token(t, nil, "second")
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			if _, err := v.Verify(context.Background(), second); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if got := f.count.Load(); got != 2 {
		t.Fatalf("rotation fetches=%d want2", got)
	}
	clock = clock.Add(accessRefreshInterval)
	for _, n := range []int{8, 16} {
		for i := range n {
			token := f.token(t, nil, strings.Repeat("unknown", i+1))
			wg.Go(func() {
				if _, err := v.Verify(context.Background(), token); err == nil {
					t.Error("unknown key accepted")
				}
			})
		}
		wg.Wait()
	}
	if got := f.count.Load(); got != 3 {
		t.Fatalf("unknown waves fetches=%d want3", got)
	}
}

func TestAccessKeysHTTPBounds(t *testing.T) {
	f := newAccessIssuerFixture(t)
	key := f.keys.Keys[0]
	for _, kind := range []string{"body", "keys", "redirect", "empty", "private", "status"} {
		t.Run(kind, func(t *testing.T) {
			var hits atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				switch kind {
				case "body":
					_, _ = w.Write([]byte(strings.Repeat(" ", 64*1024+1)))
				case "keys":
					keys := make([]jose.JSONWebKey, 9)
					for i := range keys {
						keys[i] = key
					}
					_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: keys})
				case "redirect":
					http.Redirect(w, r, "https://evil.example/keys", http.StatusFound)
				case "empty":
					_, _ = w.Write([]byte(`{"keys":[]}`))
				case "private":
					private := key
					private.Key = f.key
					_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{private}})
				case "status":
					w.WriteHeader(http.StatusServiceUnavailable)
				}
			}))
			defer server.Close()
			v, err := newAccessVerifier(AccessConfig{Issuer: fixtureIssuer, Audience: "app", AllowedEmails: []string{"human@example.com"}}, accessFixtureTransport{server.URL, http.DefaultTransport}, time.Now)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := v.Verify(context.Background(), f.token(t, nil, "first")); err == nil {
				t.Fatal("accepted invalid key response")
			}
			if hits.Load() != 1 {
				t.Fatalf("followed redirect or repeated fetch: %d", hits.Load())
			}
		})
	}
}

func TestAccessKeysCancellationAndExpiry(t *testing.T) {
	f := newAccessIssuerFixture(t)
	v := f.verifier(t, time.Now)
	token := f.token(t, nil, "first")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := v.Verify(ctx, token); err == nil {
		t.Fatal("accepted canceled request")
	}
	if f.count.Load() != 0 {
		t.Fatal("canceled request fetched")
	}
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done() }))
	defer server.Close()
	v, err := newAccessVerifier(AccessConfig{Issuer: fixtureIssuer, Audience: "app", AllowedEmails: []string{"human@example.com"}}, accessFixtureTransport{server.URL, http.DefaultTransport}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := v.Verify(ctx, token); done <- err }()
	<-started
	waiting, cancelWaiting := context.WithCancel(context.Background())
	cancelWaiting()
	if _, err := v.Verify(waiting, token); err == nil {
		t.Fatal("accepted canceled waiter")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("accepted canceled fetch")
		}
	case <-time.After(time.Second):
		t.Fatal("fetch ignored cancellation")
	}
	clock := time.Now()
	v = f.verifier(t, func() time.Time { return clock })
	if _, err := v.Verify(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	f.server.Close()
	clock = clock.Add(accessKeyTTL)
	if _, err := v.Verify(context.Background(), token); err == nil {
		t.Fatal("expired cache authorized while issuer unavailable")
	}
}

func TestAccessVerifierRefusesForgedAndWrongAlgorithm(t *testing.T) {
	f := newAccessIssuerFixture(t)
	v := f.verifier(t, time.Now)
	valid := f.token(t, nil, "first")
	parts := strings.Split(valid, ".")
	parts[1] = "eyJlbWFpbCI6Imh1bWFuQGV4YW1wbGUuY29tIn0"
	if _, err := v.Verify(context.Background(), strings.Join(parts, ".")); err == nil {
		t.Fatal("forgery accepted")
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: []byte(strings.Repeat("s", 32))}, (&jose.SignerOptions{}).WithHeader("kid", "first"))
	if err != nil {
		t.Fatal(err)
	}
	obj, err := signer.Sign([]byte(`{"iss":"` + fixtureIssuer + `"}`))
	if err != nil {
		t.Fatal(err)
	}
	token, err := obj.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Verify(context.Background(), token); err == nil {
		t.Fatal("wrong algorithm accepted")
	}
}

func TestAccessKeysIgnoresTokenSuppliedURLs(t *testing.T) {
	f := newAccessIssuerFixture(t)
	v := f.verifier(t, time.Now)
	var attackerHits atomic.Int32
	attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attackerHits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer attacker.Close()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: f.key}, (&jose.SignerOptions{}).WithHeader("kid", "first").WithHeader("jku", attacker.URL).WithHeader("x5u", attacker.URL))
	if err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{"iss": fixtureIssuer, "aud": []string{"app"}, "sub": "human", "email": "human@example.com", "type": "app", "exp": time.Now().Add(time.Hour).Unix()}
	data, err := json.Marshal(payload)
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
	if _, err := v.Verify(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if attackerHits.Load() != 0 || f.count.Load() != 1 {
		t.Fatalf("key URL pin violated: attacker=%d issuer=%d", attackerHits.Load(), f.count.Load())
	}
}

func TestAccessKeysUnavailableFetchIsRateLimited(t *testing.T) {
	f := newAccessIssuerFixture(t)
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	v, err := newAccessVerifier(AccessConfig{Issuer: fixtureIssuer, Audience: "app", AllowedEmails: []string{"human@example.com"}}, accessFixtureTransport{server.URL, http.DefaultTransport}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	token := f.token(t, nil, "first")
	for _, n := range []int{8, 16} {
		var wg sync.WaitGroup
		for range n {
			wg.Go(func() {
				if _, err := v.Verify(context.Background(), token); err == nil {
					t.Error("unavailable keys accepted")
				}
			})
		}
		wg.Wait()
	}
	if hits.Load() != 1 {
		t.Fatalf("failure waves fetch count=%d want1", hits.Load())
	}
}
