package broadcast

import (
	"encoding/json"
	"image/color"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

func TestHex(t *testing.T) {
	cases := []struct {
		c    color.Color
		want string
	}{
		{nil, ""},
		{color.RGBA{0x1e, 0x1f, 0x29, 0xff}, "#1e1f29"},
		{color.RGBA64{0xffff, 0x8080, 0, 0xffff}, "#ff8000"},
	}
	for _, c := range cases {
		if got := Hex(c.c); got != c.want {
			t.Errorf("Hex(%v) = %q, want %q", c.c, got, c.want)
		}
	}
}

func TestServerSendsThemeBeforeFirstFrame(t *testing.T) {
	theme := Theme{Foreground: "#eeeeee", Background: "#101010"}
	theme.ANSI[1] = "#ff5555"
	h := NewHub(HubOptions{})
	t.Cleanup(func() { h.Close(nil) })
	srv := NewServer(ServerOptions{Token: testToken, Hub: h, Theme: func() Theme { return theme }})
	ts := httpServer(t, srv)
	offer(h, live(t, "frame"), terminal.FramePublic)
	_, events := openEvents(t, ts.URL)
	var first sseEvent
	for first = events.next(); first.comment; first = events.next() {
	}
	if first.name != "theme" {
		t.Fatalf("first event %q, want theme", first.name)
	}
	var got Theme
	if err := json.Unmarshal([]byte(first.data), &got); err != nil {
		t.Fatal(err)
	}
	if got != theme {
		t.Fatalf("theme %+v, want %+v", got, theme)
	}
	if next := events.next(); next.name != "frame" {
		t.Fatalf("after theme: %q", next.name)
	}
}

func TestServerWithoutThemeSendsNone(t *testing.T) {
	h, srv := testServer(t, HubOptions{}, time.Hour)
	ts := httpServer(t, srv)
	offer(h, live(t, "frame"), terminal.FramePublic)
	_, events := openEvents(t, ts.URL)
	ev := events.next()
	for ev.comment {
		ev = events.next()
	}
	if ev.name != "frame" {
		t.Fatalf("first event %q, want frame", ev.name)
	}
}

func TestServerServesFontsUnderToken(t *testing.T) {
	_, srv := testServer(t, HubOptions{}, 0)
	for _, face := range []string{"Regular", "Bold", "Italic", "BoldItalic"} {
		path := "/" + testToken + "/fonts/JetBrainsMono-" + face + ".woff2"
		rec := serve(t, srv, http.MethodGet, path)
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "font/woff2" || !strings.HasPrefix(rec.Body.String(), "wOF2") {
			t.Fatalf("GET %s = %d %q", path, rec.Code, rec.Header().Get("Content-Type"))
		}
		if rec := serve(t, srv, http.MethodGet, "/wrong/fonts/JetBrainsMono-"+face+".woff2"); rec.Code != http.StatusNotFound {
			t.Fatalf("font served without the token: %d", rec.Code)
		}
	}
	symbols := "/" + testToken + "/fonts/NotoSansSymbols2-Couch.woff"
	if rec := serve(t, srv, http.MethodGet, symbols); rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "font/woff" || !strings.HasPrefix(rec.Body.String(), "wOFF") {
		t.Fatalf("GET %s = %d %q", symbols, rec.Code, rec.Header().Get("Content-Type"))
	}
	if rec := serve(t, srv, http.MethodGet, "/wrong/fonts/NotoSansSymbols2-Couch.woff"); rec.Code != http.StatusNotFound {
		t.Fatalf("symbol font served without the token: %d", rec.Code)
	}
	for _, unlisted := range []string{"OFL.txt", "OFL-NotoSansSymbols2.txt", "symbols.txt", "symbols.py"} {
		if rec := serve(t, srv, http.MethodGet, "/"+testToken+"/fonts/"+unlisted); rec.Code != http.StatusNotFound {
			t.Fatalf("unlisted vendor file %s served: %d", unlisted, rec.Code)
		}
	}
}
