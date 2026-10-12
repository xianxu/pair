package assets

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// cssCase is one entry of tests/webcast/css_urls.json, the corpus the
// viewer's css.js is held to as well: which spans of a stylesheet are URLs.
type cssCase struct {
	Name string   `json:"name"`
	CSS  string   `json:"css"`
	Base string   `json:"base"`
	URLs []string `json:"urls"`
}

func TestRewriteCSSCorpus(t *testing.T) {
	raw, err := os.ReadFile("../../../../tests/webcast/css_urls.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []cssCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		got := []string{}
		RewriteCSS(c.CSS, c.Base, func(abs string) string { got = append(got, abs); return abs })
		if !reflect.DeepEqual(got, c.URLs) {
			t.Errorf("%s: urls %q, want %q", c.Name, got, c.URLs)
		}
	}
}

func TestRewriteCSSExactWhenUnchanged(t *testing.T) {
	css := "body{background:url(http://x/bg.png)} @import 'http://x/i.css'; a{b:image-set(\"http://x/a.png\" 1x)}"
	if got := RewriteCSS(css, "http://x/", func(abs string) string { return abs }); got != css {
		t.Errorf("an unchanged sheet must keep its bytes:\n%s", got)
	}
}

func TestRewriteCSSAbsolutizes(t *testing.T) {
	css := `/* url(c.png) don't */ a{background:url(a.png)} @import "i.css"; b{c:-webkit-image-set('p.png' 1x)} d{content:"x\"y"}`
	want := `/* url(c.png) don't */ a{background:url("http://x/d/a.png")} @import "http://x/d/i.css"; b{c:-webkit-image-set("http://x/d/p.png" 1x)} d{content:"x\"y"}`
	if got := RewriteCSS(css, "http://x/d/s.css", func(abs string) string { return abs }); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestRewriteCSSToHash(t *testing.T) {
	css := `a{background:url(http://x/a.png)} b{background:url("http://x/miss.png")}`
	to := func(abs string) string {
		if abs == "http://x/a.png" {
			return "ab12"
		}
		return abs
	}
	want := `a{background:url("ab12")} b{background:url("http://x/miss.png")}`
	if got := RewriteCSS(css, "", to); got != want {
		t.Errorf("got %s", got)
	}
}

func TestRewriteCSSUnterminated(t *testing.T) {
	// Broken sheets must not panic or loop; what can't be parsed is copied.
	for _, css := range []string{`a{b:url(`, `a{b:url("x`, `/* never closed`, `@import "x`, `a{b:image-set("x"`, `"`, `url(`} {
		RewriteCSS(css, "http://x/", func(abs string) string { return "h" })
	}
}
