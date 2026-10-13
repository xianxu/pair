package webcast

import (
	"context"
	"encoding/json"
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
	locks := &broadcast.LinkLocks{}
	if err := locks.Mint(broadcast.LinkView, "tok", 4, nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(locks.End)
	lock := locks.LockFor("tok")
	viewer := NewServer("tok", NewFakeBrowser(), time.Second, hub, nil).WithLocks(locks)
	viewer.Assets().Bind(store)
	host := broadcast.NewHost(broadcast.HostConfig{Tunnel: broadcast.LocalOnly{}, ProbeTimeout: 5 * time.Second})
	mount, err := host.Mount(context.Background(), viewer, "tok/")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mount.Unmount)
	link := mount.Base() + "/tok/"

	waitNav := func(what, expr, want string) {
		t.Helper()
		deadline := time.Now().Add(20 * time.Second)
		got := ""
		for time.Now().Before(deadline) {
			var res struct {
				Result struct{ Value json.RawMessage }
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := page.conn.Call(ctx, page.session, "Runtime.evaluate", map[string]any{"expression": expr, "awaitPromise": true, "returnByValue": true}, &res)
			cancel()
			if err == nil && json.Unmarshal(res.Result.Value, &got) == nil && got == want {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Fatalf("%s: got %.300q, want %q", what, got, want)
	}
	navigate := func(url string) {
		page.call(t, page.session, "Page.navigate", map[string]any{"url": url}, nil)
	}
	onLockPage := `document.getElementById('lock') ? 'lock' : (document.getElementById('page') ? 'viewer' : 'loading')`

	// Typed entry: a wrong code is refused on the page; the right one opens.
	navigate(link)
	waitNav("the lock page", onLockPage, "lock")
	submit := func(code, name string) {
		waitNav("the submit", fmt.Sprintf(`(() => {
			document.getElementById('code').value = %q;
			document.getElementById('name').value = %q;
			document.getElementById('lock').requestSubmit();
			return 'ok';
		})()`, code, name), "ok")
	}
	wrong := []byte(lock.Code())
	wrong[0] = '0' + (wrong[0]-'0'+1)%10
	submit(string(wrong), "Ada")
	waitNav("the refusal", `document.getElementById('message').textContent`, "That code isn't right.")
	submit(lock.Code(), "Ada")
	waitNav("the viewer after typing the code", onLockPage, "viewer")
	waitNav("the name kept for notes", `localStorage.getItem('couch.viewerName')`, "Ada")
	waitNav("the drawn page", `String((document.getElementById('page').contentDocument?.body?.childElementCount || 0) > 0)`, "true")
	waitNav("recorded assets loaded with the session", `(() => {
		const got = performance.getEntriesByType('resource').filter((e) => e.name.includes('/tok/assets/'));
		return got.length > 0 && got.every((e) => e.responseStatus === 200) ? 'ok' : 'statuses ' + got.map((e) => e.responseStatus).join(',');
	})()`, "ok")

	// A rotation sends this viewer back to the lock page.
	if _, err := lock.Rotate(); err != nil {
		t.Fatal(err)
	}
	waitNav("the lock page after rotation", onLockPage, "lock")

	// URL#code with the name remembered: in by itself, the code gone from the URL.
	// From elsewhere, as a viewer opens it: on the same document a new
	// fragment alone would not reload the page.
	navigate("about:blank")
	waitNav("a blank page", `location.href`, "about:blank")
	navigate(link + "#" + lock.Code())
	waitNav("the viewer from the fragment", onLockPage, "viewer")
	waitNav("no code left in the URL", `location.hash`, "")
}
