package remotesession

import (
	"sync"
	"testing"
	"time"
)

func sessionIdentity(now time.Time) AccessIdentity {
	return AccessIdentity{issuer: "https://team.cloudflareaccess.com", subject: "sub", audience: "aud", email: "user@example.com", expires: now.Add(48 * time.Hour)}
}
func TestRemoteSessionsAccessLifetimeAndRevocation(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	id := sessionIdentity(now)
	s := newSessions(1)
	c, err := s.Handshake(id, now)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := s.Admit(c.Cookie, c.CSRF, id, now.Add(24*time.Hour), true)
	if err != nil {
		t.Fatalf("artificial absolute/idle expiry: %v", err)
	}
	select {
	case <-ref.done:
		t.Fatal("live session revoked")
	default:
	}
	fresh := id
	fresh.expires = id.expires.Add(time.Hour)
	if _, err := s.Admit(c.Cookie, c.CSRF, fresh, id.expires, true); err == nil {
		t.Fatal("new assertion extended old session")
	}
	select {
	case <-ref.done:
	default:
		t.Fatal("expired stream not cancelled")
	}
	renewed, err := s.Handshake(fresh, id.expires)
	if err != nil {
		t.Fatal("silent fresh handshake refused", err)
	}
	active, err := s.Admit(renewed.Cookie, renewed.CSRF, fresh, id.expires, true)
	if err != nil {
		t.Fatal(err)
	}
	s.Revoke()
	if _, err := s.Admit(renewed.Cookie, renewed.CSRF, fresh, id.expires, true); err == nil {
		t.Fatal("old epoch session survived")
	}
	select {
	case <-active.done:
	default:
		t.Fatal("revoked stream remains live")
	}
	if _, err := s.Handshake(fresh, id.expires); err == nil {
		t.Fatal("revoked table admitted fresh session")
	}
}
func TestRemoteSessionsConcurrentCap(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	s := newSessions(2)
	id := sessionIdentity(now)
	var wg sync.WaitGroup
	accepted := make(chan sessionCredentials, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if c, err := s.Handshake(id, now); err == nil {
				accepted <- c
			}
		}()
	}
	wg.Wait()
	close(accepted)
	count := 0
	for c := range accepted {
		count++
		if c.Cookie == c.CSRF {
			t.Fatal("cookie reused as CSRF")
		}
	}
	if count != 4 {
		t.Fatalf("concurrent cap admitted %d", count)
	}
}
func TestRemoteSessionsPrincipalCSRFAndLogout(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	s := newSessions(3)
	id := sessionIdentity(now)
	c, err := s.Handshake(id, now)
	if err != nil {
		t.Fatal(err)
	}
	other := id
	other.subject = "other"
	if _, err := s.Admit(c.Cookie, c.CSRF, other, now, true); err == nil {
		t.Fatal("foreign principal")
	}
	if _, err := s.Admit(c.Cookie, "", id, now, true); err == nil {
		t.Fatal("missing CSRF")
	}
	if _, err := s.Admit(c.Cookie, "", id, now, false); err != nil {
		t.Fatal("read refused", err)
	}
	ref, err := s.Admit(c.Cookie, c.CSRF, id, now, true)
	if err != nil {
		t.Fatal(err)
	}
	s.Logout(c.Cookie)
	select {
	case <-ref.done:
	default:
		t.Fatal("logout did not revoke")
	}
	if _, err := s.Admit(c.Cookie, c.CSRF, id, now, true); err == nil {
		t.Fatal("logged out cookie accepted")
	}
}
