package remotesession

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"
)

// Credentials are unique across both cookies and CSRF tokens, without relying
// on OS entropy or the scheduling order of the callers.
func sequenceCredentials() func() (string, error) {
	var next atomic.Uint64
	return func() (string, error) {
		var raw [32]byte
		binary.BigEndian.PutUint64(raw[:8], next.Add(1))
		return base64.RawURLEncoding.EncodeToString(raw[:]), nil
	}
}

func TestRemoteSessionsCredentialFailureDoesNotReserveCapacity(t *testing.T) {
	for _, failAt := range []int{1, 2} {
		t.Run(fmt.Sprint(failAt), func(t *testing.T) {
			s := newSessions(1)
			produce := sequenceCredentials()
			calls := 0
			failure := errors.New("entropy unavailable")
			s.credential = func() (string, error) {
				calls++
				if calls == failAt {
					return "", failure
				}
				return produce()
			}
			now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
			if _, err := s.Handshake(sessionIdentity(now), now); !errors.Is(err, failure) {
				t.Fatalf("error = %v", err)
			}
			if len(s.entries) != 0 {
				t.Fatal("failed credential production reserved a slot")
			}
		})
	}
}

func sessionSignalClosed(done <-chan struct{}) bool {
	select {
	case <-done:
		return true
	default:
		return false
	}
}

func TestRemoteSessionsHeldHandshakeBoundaries(t *testing.T) {
	for _, boundary := range []string{"revoke", "logout", "expiry"} {
		t.Run(boundary, func(t *testing.T) {
			now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
			s := newSessions(7)
			s.credential = sequenceCredentials()
			oldID := sessionIdentity(now)
			oldID.expires = now.Add(time.Minute)
			var old []sessionCredentials
			var refs []sessionRef
			for i := 0; i < 4; i++ {
				c, err := s.Handshake(oldID, now)
				if err != nil {
					t.Fatal(err)
				}
				ref, err := s.Admit(c.Cookie, c.CSRF, oldID, now, true)
				if err != nil {
					t.Fatal(err)
				}
				old = append(old, c)
				refs = append(refs, ref)
			}
			held, release := make(chan struct{}), make(chan struct{})
			defer close(release)
			produce := s.credential
			var calls atomic.Int64
			s.credential = func() (string, error) {
				c, err := produce()
				if calls.Add(1) == 2 {
					close(held)
					<-release
				}
				return c, err
			}
			type result struct {
				c   sessionCredentials
				err error
			}
			resultCh := make(chan result, 1)
			admissionTime := now
			if boundary == "expiry" {
				admissionTime = oldID.expires
			}
			freshID := sessionIdentity(admissionTime)
			go func() { c, err := s.Handshake(freshID, admissionTime); resultCh <- result{c, err} }()
			<-held // Both credentials exist; the session-table mutex is not yet taken.
			switch boundary {
			case "revoke":
				s.Revoke()
			case "logout":
				s.Logout(old[0].Cookie)
			case "expiry":
				// A separate admission reclaims all four expired entries while the first
				// handshake is held outside the commit boundary.
				if _, err := s.Handshake(freshID, admissionTime); err != nil {
					t.Fatal(err)
				}
			}
			s.mu.Lock()
			ownedBefore := len(s.entries)
			s.mu.Unlock()
			wantBefore := 3
			if boundary == "revoke" {
				wantBefore = 0
			}
			if boundary == "expiry" {
				wantBefore = 1
			}
			if ownedBefore != wantBefore {
				t.Fatalf("owned while held = %d, want %d", ownedBefore, wantBefore)
			}
			release <- struct{}{}
			got := <-resultCh
			if boundary == "revoke" {
				if !errors.Is(got.err, errSession) {
					t.Fatalf("post-revoke handshake = %v", got.err)
				}
			} else {
				if got.err != nil {
					t.Fatal(got.err)
				}
				if _, err := s.Admit(got.c.Cookie, got.c.CSRF, freshID, admissionTime, true); err != nil {
					t.Fatalf("fresh grant refused: %v", err)
				}
			}
			for i, c := range old {
				retired := boundary != "logout" || i == 0
				if sessionSignalClosed(refs[i].done) != retired {
					t.Fatalf("old stream %d cancellation mismatch", i)
				}
				_, err := s.Admit(c.Cookie, c.CSRF, freshID, admissionTime, true)
				if (err != nil) != retired {
					t.Fatalf("old cookie %d: retired=%v err=%v", i, retired, err)
				}
			}
			s.mu.Lock()
			ownedAfter := len(s.entries)
			s.mu.Unlock()
			wantAfter := wantBefore
			if boundary != "revoke" {
				wantAfter++
			}
			if ownedAfter != wantAfter {
				t.Fatalf("owned after release = %d, want %d", ownedAfter, wantAfter)
			}
		})
	}
}

func TestRemoteSessionsRendezvousCapacityWaves(t *testing.T) {
	for _, wave := range []int{8, 16} {
		t.Run(fmt.Sprint(wave), func(t *testing.T) {
			now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
			s := newSessions(8)
			produce := sequenceCredentials()
			arrived := make(chan struct{}, wave)
			release := make(chan struct{})
			var calls atomic.Int64
			s.credential = func() (string, error) {
				if calls.Add(1) <= int64(wave) {
					arrived <- struct{}{}
					<-release
				}
				return produce()
			}
			results := make(chan error, wave)
			for i := 0; i < wave; i++ {
				go func() { _, err := s.Handshake(sessionIdentity(now), now); results <- err }()
			}
			for i := 0; i < wave; i++ {
				<-arrived
			}
			s.mu.Lock()
			before := len(s.entries)
			s.mu.Unlock()
			if before != 0 {
				t.Fatalf("held wave owns %d slots", before)
			}
			close(release)
			accepted := 0
			for i := 0; i < wave; i++ {
				err := <-results
				if err == nil {
					accepted++
				} else if !errors.Is(err, errCapacity) {
					t.Fatal(err)
				}
			}
			s.mu.Lock()
			owned := len(s.entries)
			s.mu.Unlock()
			if accepted != 4 || owned != 4 {
				t.Fatalf("accepted=%d owned=%d, want 4 each", accepted, owned)
			}
		})
	}
}

// The oracle stores only observable credentials, expiry and ownership, rather
// than calling production validity/retirement helpers or copying its hash map.
func TestRemoteSessionsGeneratedLifecycleSequences(t *testing.T) {
	for seed := int64(1); seed <= 24; seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			rng := rand.New(rand.NewSource(seed))
			now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
			s := newSessions(uint64(seed))
			s.credential = sequenceCredentials()
			type ownedSession struct {
				credentials sessionCredentials
				expires     time.Time
				owned       bool
				done        <-chan struct{}
			}
			var model []*ownedSession
			revoked := false
			revokeStep := 64 + rng.Intn(64)
			count := func() int {
				n := 0
				for _, entry := range model {
					if entry.owned {
						n++
					}
				}
				return n
			}
			assertOwned := func(step int) {
				t.Helper()
				s.mu.Lock()
				actual := len(s.entries)
				s.mu.Unlock()
				expected := count()
				if actual != expected || actual > 4 {
					t.Fatalf("seed=%d step=%d owned=%d model=%d", seed, step, actual, expected)
				}
				for i, entry := range model {
					if sessionSignalClosed(entry.done) == entry.owned {
						t.Fatalf("seed=%d step=%d stream=%d owned=%v cancellation mismatch", seed, step, i, entry.owned)
					}
				}
			}
			for step := 0; step < 128; step++ {
				op := rng.Intn(5)
				if step == revokeStep {
					op = 5
				}
				switch op {
				case 0, 1: // Fresh valid Access assertion, including after old sessions expire.
					id := sessionIdentity(now)
					id.expires = now.Add(time.Duration(1+rng.Intn(8)) * time.Minute)
					if !revoked {
						for _, entry := range model {
							if !now.Before(entry.expires) {
								entry.owned = false
							}
						}
					}
					want := error(nil)
					if revoked {
						want = errSession
					} else if count() == 4 {
						want = errCapacity
					}
					c, err := s.Handshake(id, now)
					if !errors.Is(err, want) {
						t.Fatalf("step=%d handshake error=%v want=%v", step, err, want)
					}
					if err == nil {
						ref, err := s.Admit(c.Cookie, c.CSRF, id, now, true)
						if err != nil {
							t.Fatal(err)
						}
						model = append(model, &ownedSession{c, id.expires, true, ref.done})
					}
				case 2:
					if len(model) > 0 {
						entry := model[rng.Intn(len(model))]
						id := sessionIdentity(now) // A renewed assertion cannot extend the stored grant.
						validAccess := rng.Intn(3) != 0
						if !validAccess {
							id.expires = now
						}
						wantGrant := entry.owned && !revoked && now.Before(entry.expires) && validAccess
						_, err := s.Admit(entry.credentials.Cookie, entry.credentials.CSRF, id, now, true)
						if (err == nil) != wantGrant {
							t.Fatalf("step=%d admit=%v wantGrant=%v", step, err, wantGrant)
						}
						if entry.owned && !now.Before(entry.expires) {
							entry.owned = false
						}
					}
				case 3:
					if len(model) > 0 {
						entry := model[rng.Intn(len(model))]
						s.Logout(entry.credentials.Cookie)
						entry.owned = false
					}
				case 4:
					now = now.Add(time.Duration(1+rng.Intn(5)) * time.Minute)
				case 5:
					s.Revoke()
					revoked = true
					for _, entry := range model {
						entry.owned = false
					}
				}
				assertOwned(step)
				// Probe every historical credential. This checks dead-cookie permanence,
				// expiry despite refreshed Access, and fresh valid grants after reclamation.
				for _, entry := range model {
					wantGrant := entry.owned && !revoked && now.Before(entry.expires)
					_, err := s.Admit(entry.credentials.Cookie, entry.credentials.CSRF, sessionIdentity(now), now, true)
					if (err == nil) != wantGrant {
						t.Fatalf("step=%d historical admit=%v wantGrant=%v", step, err, wantGrant)
					}
					if entry.owned && !now.Before(entry.expires) {
						entry.owned = false
					}
				}
				assertOwned(step)
			}
		})
	}
}
