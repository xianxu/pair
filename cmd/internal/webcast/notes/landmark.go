package notes

import (
	"slices"
	"strings"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// The container rule table: what makes an element a meaningful region of a
// page (ARIA landmarks, roles, and layout boxes recognized by class).
var (
	containerTags = set("header", "nav", "main", "aside", "footer", "form", "section", "article", "dialog",
		"fieldset", "figure", "table", "tr", "ul", "ol", "dl", "li", "details", "menu", "search")
	containerRoles = set("banner", "navigation", "main", "complementary", "contentinfo", "form", "search",
		"region", "dialog", "alertdialog", "article", "list", "listitem", "table", "row", "grid", "group",
		"toolbar", "menu", "menubar", "tablist", "tabpanel", "figure")
	boxClassPrefixes = []string{"card", "panel", "modal", "dialog", "popover", "toolbar", "navbar", "sidebar", "drawer", "tile"}
)

// maxPath is how many steps an anchor's path keeps, the innermost.
const maxPath = 6

// pathOf is an anchor's path (#488): the meaningful containers above n,
// outermost first, then n itself, each named by its tag and its id or role
// (main, article, p#first). The innermost maxPath steps.
func pathOf(n *recording.DOMNode) []string {
	var out []string
	for e := n; e != nil; e = e.ParentElement() {
		if (e.N == "body" || e.N == "html") && treeRoot(e).Host == nil {
			break
		}
		if e == n || isContainer(e) {
			out = append(out, stepName(e))
		}
	}
	slices.Reverse(out)
	return out[max(0, len(out)-maxPath):]
}

func stepName(e *recording.DOMNode) string {
	switch {
	case e.A["id"] != "" && len(e.A["id"]) <= 40:
		return e.N + "#" + e.A["id"]
	case e.A["role"] != "" && len(e.A["role"]) <= 40:
		return e.N + "[" + e.A["role"] + "]"
	}
	return e.N
}

// contextBudget is how much of the landmark's text a context carries.
const contextBudget = 500

// landmark is the nearest meaningful container at or above n (through
// shadow hosts and frame owners), below the top document's body. With none,
// it is n's parent element (body at most) and meaningful is false: good for
// text, not for snapping a crop to.
func landmark(n *recording.DOMNode) (land *recording.DOMNode, meaningful bool) {
	for e := n; e != nil; e = e.ParentElement() {
		if (e.N == "body" || e.N == "html") && treeRoot(e).Host == nil {
			break
		}
		if isContainer(e) {
			return e, true
		}
	}
	if p := n.ParentElement(); p != nil && p.N != "html" {
		return p, false
	}
	return n, false
}

func isContainer(e *recording.DOMNode) bool {
	if containerTags[e.N] || containerRoles[e.A["role"]] || e.A["aria-modal"] == "true" {
		return true
	}
	return hasElementKid(e) && boxClass(e.A["class"])
}

// boxClass recognizes a layout box by its classes: a named one (card,
// panel, …), or Tailwind's rounded corners with a border or a shadow.
func boxClass(class string) bool {
	var rounded, edged bool
	for _, tok := range strings.Fields(strings.ToLower(class)) {
		tok = tok[strings.LastIndexByte(tok, ':')+1:] // md:shadow → shadow
		for _, p := range boxClassPrefixes {
			if strings.HasPrefix(tok, p) {
				return true
			}
		}
		rounded = rounded || strings.HasPrefix(tok, "rounded")
		edged = edged || tok == "border" || tok == "shadow" || strings.HasPrefix(tok, "shadow-")
	}
	return rounded && edged
}

func hasElementKid(e *recording.DOMNode) bool {
	for _, k := range e.Kids {
		if k.Y == recording.YEl {
			return true
		}
	}
	return false
}

// window is text whitespace-collapsed and cut to budget runes, centred on
// the first occurrence of focus so the anchor's own text stays in it.
func window(text, focus string, budget int) string {
	r := []rune(strings.Join(strings.Fields(text), " "))
	if len(r) <= budget {
		return string(r)
	}
	start := 0
	if focus = strings.Join(strings.Fields(focus), " "); focus != "" {
		if i := strings.Index(string(r), focus); i >= 0 {
			at, n := len([]rune(string(r)[:i])), len([]rune(focus))
			start = max(0, min(at+n/2-budget/2, len(r)-budget))
			if n >= budget {
				start = at
			}
		}
	}
	return string(r[start : start+budget])
}

func set(xs ...string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}
