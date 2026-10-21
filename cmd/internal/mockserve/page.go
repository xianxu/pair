package mockserve

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Bootstrap identifies the actual document and the scan observed before its read.
type Bootstrap struct {
	Version  string `json:"version"`
	Screen   string `json:"screen"`
	Revision string `json:"revision"`
}

// ReloadScript is served as a separate same-origin script by Server.
//
//go:embed reload.js
var ReloadScript []byte

// ScreenFromPath validates a normalized version-relative HTML file route.
func ScreenFromPath(name string) (string, error) {
	if name == "" || strings.HasPrefix(name, "/") || path.Clean(name) != name || strings.ContainsAny(name, "\\\x00?#") || !strings.HasSuffix(name, ".html") {
		return "", fmt.Errorf("invalid screen path %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == "." || part == ".." || part == ".html" {
			return "", fmt.Errorf("invalid screen path %q", name)
		}
	}
	return strings.TrimSuffix(name, ".html"), nil
}

// TransformPage normalizes HTML and installs authoritative metadata before scripts.
func TransformPage(input []byte, boot Bootstrap) ([]byte, error) {
	root, err := html.Parse(bytes.NewReader(input))
	if err != nil {
		return nil, err
	}
	var htmlRoot, head *html.Node
	var clean func(*html.Node)
	clean = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "html" {
				htmlRoot = n
			}
			if n.Data == "head" {
				head = n
			}
			attrs := n.Attr[:0]
			for _, a := range n.Attr {
				if a.Key != "data-mock-version" && a.Key != "data-mock-screen" {
					attrs = append(attrs, a)
				}
			}
			n.Attr = attrs
		}
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			owned := false
			for _, a := range c.Attr {
				if a.Key == "id" && a.Val == "couch-mock-bootstrap" || c.Data == "script" && a.Key == "src" && a.Val == "/__couch_mock/reload.js" {
					owned = true
				}
			}
			if owned {
				n.RemoveChild(c)
			} else {
				clean(c)
			}
			c = next
		}
	}
	clean(root)
	if htmlRoot == nil || head == nil {
		return nil, fmt.Errorf("HTML parser produced no document head")
	}
	htmlRoot.Attr = append(htmlRoot.Attr, html.Attribute{Key: "data-mock-version", Val: boot.Version}, html.Attribute{Key: "data-mock-screen", Val: boot.Screen})
	encoded, err := json.Marshal(boot)
	if err != nil {
		return nil, err
	}
	bootstrap := &html.Node{Type: html.ElementNode, Data: "script", DataAtom: atom.Script, Attr: []html.Attribute{{Key: "id", Val: "couch-mock-bootstrap"}, {Key: "type", Val: "application/json"}}}
	bootstrap.AppendChild(&html.Node{Type: html.TextNode, Data: string(encoded)})
	reload := &html.Node{Type: html.ElementNode, Data: "script", DataAtom: atom.Script, Attr: []html.Attribute{{Key: "src", Val: "/__couch_mock/reload.js"}}}
	first := head.FirstChild
	head.InsertBefore(bootstrap, first)
	head.InsertBefore(reload, first)
	var out bytes.Buffer
	if err := html.Render(&out, root); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
