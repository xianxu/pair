package recording

import "testing"

func TestRenderedText(t *testing.T) {
	txt := func(s string) *DOMNode { return &DOMNode{Y: YText, V: s} }
	el := func(tag string, attrs []string, kids ...*DOMNode) *DOMNode {
		e := &DOMNode{Y: YEl, N: tag, A: map[string]string{}, Kids: kids}
		for i := 0; i+1 < len(attrs); i += 2 {
			e.A[attrs[i]] = attrs[i+1]
		}
		return e
	}
	for _, c := range []struct {
		name string
		n    *DOMNode
		want string
	}{
		{"blocks apart", el("div", nil, el("h2", nil, txt("tasks")), el("p", nil, txt("Add"))), "tasks Add"},
		{"inline runs on", el("p", nil, txt("Ship"), el("b", nil, txt("ping")), txt(" now")), "Shipping now"},
		{"cells and rows", el("table", nil, el("tr", nil, el("td", nil, txt("Week 1")), el("td", nil, txt("0 tasks"))),
			el("tr", nil, el("td", nil, txt("Week 2")))), "Week 1 0 tasks Week 2"},
		{"br", el("p", nil, txt("a"), el("br", nil), txt("b")), "a b"},
		{"list items", el("ul", nil, el("li", nil, txt("Ship onboarding flow")), el("li", nil, txt("Fix billing"))),
			"Ship onboarding flow Fix billing"},
		{"flex items", el("div", []string{"class", "md:flex gap-2"}, el("span", nil, txt("Add")), el("span", nil, txt("Ship"))), "Add Ship"},
		{"script and style", el("div", nil, txt("Reports"), el("script", nil, txt("const r=1")), el("style", nil, txt("p{}"))), "Reports"},
		{"noscript and template", el("div", nil, el("noscript", nil, txt("enable JS")), el("template", nil, txt("x")), txt("ok")), "ok"},
		{"hidden attribute", el("div", nil, el("p", []string{"hidden", ""}, txt("secret")), txt("shown")), "shown"},
		{"inline display none", el("div", nil, el("p", []string{"style", "DISPLAY: none"}, txt("secret")), txt("shown")), "shown"},
		{"inline visibility hidden", el("div", nil, el("p", []string{"style", "visibility:hidden"}, txt("secret")), txt("shown")), "shown"},
		{"tailwind hidden", el("div", nil, el("p", []string{"class", "hidden text-sm"}, txt("secret")), txt("shown")), "shown"},
		{"tailwind hidden shown at md", el("div", nil, el("p", []string{"class", "hidden md:block"}, txt("wide")), txt("shown")), "wide shown"},
		{"hover variant keeps hidden", el("div", nil, el("p", []string{"class", "hidden hover:bg-red-500"}, txt("secret")), txt("shown")), "shown"},
		{"whitespace collapses", el("p", nil, txt("  a \n\t b  ")), "a b"},
	} {
		if got := c.n.RenderedText(); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}
