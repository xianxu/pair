package keyhelp

import (
	"fmt"
	"strings"
)

// All modules owning review-local mappings; shared by composition and drift tests.
var reviewSourcePaths = []string{"nvim/review.lua", "nvim/review/comment.lua", "nvim/review/comment_float.lua"}

// Review rows have their own source identity: draft and review deliberately
// bind the same keys to different actions. Wording stays in the actual keymaps.
var reviewCatalog = []struct{ key, display string }{
	{"<CR>", "Enter (on comment; normal)"}, {"q", "q (thread float; normal)"},
	{"<Esc>", "Esc (normal)"}, {"<M-c>", "Alt+c"},
	{"<M-a>", "Alt+a (normal)"}, {"<M-r>", "Alt+r (normal)"},
	{"<leader>a", "Leader+a (normal)"}, {"<leader>r", "Leader+r (normal)"},
	{"<M-A>", "Alt+Shift+A (normal)"}, {"<M-R>", "Alt+Shift+R (normal)"},
	{"<M-q>", "Alt+q (normal/insert; visual)"}, {"<M-CR>", "Alt+⏎ (normal/insert)"},
	{"<M-n>", "Alt+n (normal)"}, {"<M-N>", "Alt+Shift+N (normal)"},
	{"]m", "]m (normal)"}, {"[m", "[m (normal)"},
	{"define_key", "Alt+Shift+D (visual)"},
	{"<C-w>d", "Ctrl+w d (normal)"}, {"gl", "gl (normal)"},
	{"]d", "]d (normal)"}, {"[d", "[d (normal)"},
}

func reviewKey(km NvimKeymap) string {
	if km.Key != "" {
		return km.Key
	}
	return km.Raw
}
func reviewClassified(km NvimKeymap) bool {
	for _, row := range reviewCatalog {
		if row.key == reviewKey(km) {
			return true
		}
	}
	return false
}

func reviewSections(src string) ([]Section, error) {
	scan := ParseReviewKeymaps(src)
	descriptions := map[string][]string{}
	for _, km := range append(append(scan.Resolved, scan.Dynamic...), scan.Unresolved...) {
		key := reviewKey(km)
		seen := false
		for _, d := range descriptions[key] {
			if d == km.Desc {
				seen = true
			}
		}
		if !seen {
			descriptions[key] = append(descriptions[key], km.Desc)
		}
	}
	section := Section{Title: "Review buffer — local keys (normal unless marked)"}
	for _, row := range reviewCatalog {
		ds := descriptions[row.key]
		if len(ds) == 0 {
			return nil, fmt.Errorf("keyhelp: review mapping %q has no review: desc", row.key)
		}
		// No global chord identity: these local actions remain available under the
		// host's review-focus exception; they are not the global restart action.
		section.Bindings = append(section.Bindings, Binding{Key: row.display, Desc: strings.Join(ds, "; "), Context: ContextReview})
	}
	return []Section{section}, nil
}
