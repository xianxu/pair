package webcast

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/castlock"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
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

// A recorded cast publishes three links, each its own token and each
// classified as its kind; a cast without a board has nothing to point at,
// so no pointer link (pair#506).
func TestCastPublishesThreeLinks(t *testing.T) {
	c, _ := startDrivenCast(t, &recording.MemSink{})
	u := waitUpdate(t, c, "links", func(u Update) bool { return u.Link != "" && u.PointerLink != "" && u.DriveLink != "" })
	locks := map[*castlock.Lock]bool{}
	for link, want := range map[string]castLink{u.Link: linkView, u.PointerLink: linkPointer, u.DriveLink: linkDrive} {
		if got := c.server.linkOf(path.Base(link)); got != want {
			t.Errorf("%s is link %d, want %d", link, got, want)
		}
		// Each link has its own code lock (pair#495), the pointer link's too.
		locks[c.server.LockFor(path.Base(link))] = true
	}
	if len(locks) != 3 || locks[nil] {
		t.Errorf("the three links share or lack locks: %v", locks)
	}
	var labels []string
	for _, code := range c.Codes() {
		labels = append(labels, code.Link)
	}
	if strings.Join(labels, " ") != "view pointer drive" {
		t.Errorf("codes %v, want view pointer drive", labels)
	}
	plain, _ := startCast(t, &broadcast.FakeTunnel{})
	if u := waitUpdate(t, plain, "link", func(u Update) bool { return u.Link != "" }); u.PointerLink != "" {
		t.Fatalf("a cast without a board has a pointer link: %s", u.PointerLink)
	}
}

// Every link goes with the mount: a lost tunnel clears the pointer link with
// the view-only link, so Couch never copies a dead one (pair#506).
func TestCastLinksClearTogetherWhenTheMountIsLost(t *testing.T) {
	tunnel := &broadcast.FakeTunnel{}
	c, err := StartCast(context.Background(), CastConfig{
		Launch:        func(context.Context, *RecordOptions) (Browser, error) { return NewFakeBrowser(), nil },
		Host:          broadcast.NewHost(broadcast.HostConfig{Tunnel: tunnel, ProbeTimeout: 2 * time.Second}),
		Ping:          time.Second,
		RecordingsDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.End)
	waitUpdate(t, c, "links", func(u Update) bool { return u.Link != "" && u.PointerLink != "" })
	tunnel.Exit()
	if u := waitUpdate(t, c, "link lost", func(u Update) bool { return u.LinkErr != nil }); u.Link != "" || u.PointerLink != "" || u.DriveLink != "" {
		t.Fatalf("links after the mount was lost: %+v", u)
	}
}

// LinkFor is the one place a code's label becomes its link's URL: unlocking,
// the Codes menu and Couch's copies all read it (pair#506).
func TestUpdateLinkFor(t *testing.T) {
	u := Update{Link: "v/", PointerLink: "p/", DriveLink: "d/"}
	for kind, want := range map[string]string{broadcast.LinkView: "v/", broadcast.LinkPointer: "p/", broadcast.LinkDrive: "d/", "other": ""} {
		if got := u.LinkFor(kind); got != want {
			t.Errorf("LinkFor(%q) = %q, want %q", kind, got, want)
		}
	}
}
