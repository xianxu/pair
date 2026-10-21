package mockserve

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestScreenFromPath(t *testing.T) {
	for name, want := range map[string]string{"index.html": "index", "runs/detail.html": "runs/detail", "hello world.html": "hello world"} {
		got, err := ScreenFromPath(name)
		if err != nil || got != want {
			t.Errorf("%q => %q, %v", name, got, err)
		}
	}
	for _, name := range []string{"", "/index.html", "../index.html", "a/../index.html", "a//b.html", "./index.html", `a\b.html`, "index.htm", "a/.html", "a.html?x", "a\x00.html"} {
		if got, err := ScreenFromPath(name); err == nil {
			t.Errorf("accepted %q => %q", name, got)
		}
	}
}

func TestTransformPageOwnsMarksAndBootstrap(t *testing.T) {
	for _, input := range []string{
		`<h1 data-mock-version="wrong" data-mock-screen="wrong">Hello</h1><script>project()</script>`,
		`<HTML DATA-MOCK-VERSION="wrong" data-mock-version="other"><HEAD><script src="project.js"></script></HEAD><BODY data-mock-screen="wrong"></BODY></HTML>`,
		`<html><head><script id="couch-mock-bootstrap">old</script><script src="/__couch_mock/reload.js"></script></head><body></body></html>`,
		`<html><head></head><body><svg><head></head></svg><script>project()</script></body></html>`,
	} {
		boot := Bootstrap{Version: `v<&"`, Screen: "runs/detail", Revision: `</script><script>alert(1)</script>`}
		out, err := TransformPage([]byte(input), boot)
		if err != nil {
			t.Fatal(err)
		}
		root, err := html.Parse(bytes.NewReader(out))
		if err != nil {
			t.Fatal(err)
		}
		marks, boots, reloads, projectScripts := 0, 0, 0, 0
		var walk func(*html.Node)
		walk = func(n *html.Node) {
			attr := map[string]string{}
			for _, a := range n.Attr {
				attr[a.Key] = a.Val
				if a.Key == "data-mock-version" || a.Key == "data-mock-screen" {
					marks++
					if n.Data != "html" {
						t.Errorf("non-root mark: %s", n.Data)
					}
				}
			}
			if n.Data == "html" && (attr["data-mock-version"] != boot.Version || attr["data-mock-screen"] != boot.Screen) {
				t.Errorf("root attrs: %v", attr)
			}
			if n.Type == html.ElementNode && n.Data == "script" {
				switch {
				case attr["id"] == "couch-mock-bootstrap":
					boots++
					if n.Parent.Data != "head" || n.Parent.Namespace != "" || n.Parent.Parent.Data != "html" {
						t.Error("bootstrap must be in document head")
					}
					if projectScripts != 0 {
						t.Error("bootstrap after project script")
					}
					var decoded Bootstrap
					if n.FirstChild == nil {
						t.Error("missing JSON")
					} else if err := json.Unmarshal([]byte(n.FirstChild.Data), &decoded); err != nil || decoded != boot {
						t.Errorf("bootstrap %v, %v", decoded, err)
					}
				case attr["src"] == "/__couch_mock/reload.js":
					reloads++
					if projectScripts != 0 || boots != 1 {
						t.Error("reload order")
					}
				default:
					projectScripts++
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
		}
		walk(root)
		if marks != 2 || boots != 1 || reloads != 1 {
			t.Errorf("marks %d boot %d reload %d: %s", marks, boots, reloads, out)
		}
		if strings.Contains(string(out), boot.Revision) {
			t.Error("unescaped bootstrap permits script injection")
		}
	}
}
