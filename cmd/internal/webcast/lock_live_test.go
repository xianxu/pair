package webcast

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// The lock page in a real Chrome (pair#495), through the Host's gate as a
// viewer meets it: the link opens on the lock page; typing the code and a
// name lets the viewer in; a URL#code with a remembered name lets it in by
// itself and leaves no code in history; and once in, the sandboxed frame's
// recorded assets load with the session (the iframe keeps the page's
// origin). A wrong code keeps the lock page up and says so.
//
//	WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run LiveLock -v
func TestLiveLockPage(t *testing.T) {
	page := headlessPage(t)
	rec, err := recording.ReadDir(recording.Fixture("rec-blog"))
	if err != nil {
		t.Fatal(err)
	}
	store, rec := throughStore(t, rec)
	// A cast still running: the fixture up to before its stop, as
	// TestLiveViewerPage feeds it. An ended cast never reconnects.
	hub := recording.NewHub(4096)
	for _, r := range rec.Records {
		if r.Time() <= 8284 {
			hub.Put(r)
		}
	}
	t.Cleanup(func() { hub.Close() })
	lock, err := broadcast.NewLinkLock(4, nil)
	if err != nil {
		t.Fatal(err)
	}
	viewer := NewServer("tok", NewFakeBrowser(), time.Second, hub, nil).WithLocks(lock, nil)
	viewer.Assets().Bind(store)
	host := broadcast.NewHost(broadcast.HostConfig{Tunnel: broadcast.LocalOnly{}, ProbeTimeout: 5 * time.Second})
	mount, err := host.Mount(context.Background(), viewer, "tok/")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mount.Unmount)
	link := mount.Base() + "/tok/"

	navigate := func(url string) {
		page.call(t, page.session, "Page.navigate", map[string]any{"url": url}, nil)
	}
	onLockPage := `document.getElementById('lock') ? 'lock' : (document.getElementById('page') ? 'viewer' : 'loading')`

	// Typed entry: a wrong code is refused on the page; the right one opens.
	navigate(link)
	waitEval(t, page, "the lock page", onLockPage, "lock")
	submit := func(code, name string) {
		page.evalString(t, fmt.Sprintf(`(() => {
			document.getElementById('code').value = %q;
			document.getElementById('name').value = %q;
			document.getElementById('lock').requestSubmit();
			return 'ok';
		})()`, code, name))
	}
	wrong := []byte(lock.Code())
	wrong[0] = '0' + (wrong[0]-'0'+1)%10
	submit(string(wrong), "Ada")
	waitEval(t, page, "the refusal", `document.getElementById('message').textContent`, "That code isn't right.")
	submit(lock.Code(), "Ada")
	waitEval(t, page, "the viewer after typing the code", onLockPage, "viewer")
	waitEval(t, page, "the name kept for notes", `localStorage.getItem('couch.viewerName')`, "Ada")
	waitEval(t, page, "the drawn page", `String((document.getElementById('page').contentDocument?.body?.childElementCount || 0) > 0)`, "true")
	waitEval(t, page, "recorded assets loaded with the session", `(() => {
		const got = performance.getEntriesByType('resource').filter((e) => e.name.includes('/tok/assets/'));
		return got.length > 0 && got.every((e) => e.responseStatus === 200) ? 'ok' : 'statuses ' + got.map((e) => e.responseStatus).join(',');
	})()`, "ok")

	// A rotation sends this viewer back to the lock page.
	if _, err := lock.Rotate(); err != nil {
		t.Fatal(err)
	}
	waitEval(t, page, "the lock page after rotation", onLockPage, "lock")

	// URL#code with the name remembered: in by itself, the code gone from the URL.
	// From elsewhere, as a viewer opens it: on the same document a new
	// fragment alone would not reload the page.
	navigate("about:blank")
	waitEval(t, page, "a blank page", `location.href`, "about:blank")
	navigate(link + "#" + lock.Code())
	waitEval(t, page, "the viewer from the fragment", onLockPage, "viewer")
	waitEval(t, page, "no code left in the URL", `location.hash`, "")
}
