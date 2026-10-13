package webcast

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// pair#506: each link grants what the one before it does, and more.
func TestCastLinkGrants(t *testing.T) {
	for _, c := range []struct {
		link              castLink
		annotates, drives bool
	}{
		{linkNone, false, false},
		{linkView, false, false},
		{linkPointer, true, false},
		{linkDrive, true, true},
	} {
		if c.link.annotates() != c.annotates || c.link.drives() != c.drives {
			t.Errorf("link %d: annotates %v drives %v, want %v %v", c.link, c.link.annotates(), c.link.drives(), c.annotates, c.drives)
		}
	}
}

// The server tells its three links apart, and owns nothing else. A link
// that wasn't made (an empty token) never matches.
func TestLinkOfEachToken(t *testing.T) {
	f := newDriveFixture(t, time.Second)
	f.srv.WithPointer("ptr")
	for segment, want := range map[string]castLink{"tok": linkView, "ptr": linkPointer, "drv": linkDrive, "nope": linkNone, "": linkNone} {
		if got := f.srv.linkOf(segment); got != want {
			t.Errorf("linkOf(%q) = %d, want %d", segment, got, want)
		}
		if f.srv.Owns(segment) != (want != linkNone) {
			t.Errorf("Owns(%q) = %v", segment, f.srv.Owns(segment))
		}
	}
	plain := NewServer("tok", NewFakeBrowser(), time.Second, nil, nil)
	if plain.linkOf("") != linkNone || plain.linkOf("ptr") != linkNone {
		t.Fatal("a server with no pointer or drive link matched one")
	}
}

// A pointer-link stream counts with the view streams: only the drive link's
// streams can hold a seat, and couch --broadcast-list reads (view, drive).
func TestPointerStreamCountsAsView(t *testing.T) {
	f := newDriveFixture(t, time.Second)
	f.srv.WithPointer("ptr")
	srv := httptest.NewServer(f.srv)
	t.Cleanup(func() { srv.CloseClientConnections(); srv.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/ptr/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("watching by the pointer link: %v %v", resp, err)
	}
	defer resp.Body.Close()
	deadline := time.Now().Add(5 * time.Second)
	for {
		view, drive := f.srv.Viewers()
		if view == 1 && drive == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("viewers by link: view %d drive %d, want 1 0", view, drive)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
