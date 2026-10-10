package keyhelp

import (
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"strings"
	"testing"
)

func TestReviewHelpDerivesEveryMappingAndSurvivesHostLayer(t *testing.T) {
	var sources []string
	for _, path := range reviewSourcePaths {
		sources = append(sources, mustReadTreeSource(t, path))
	}
	src := strings.Join(sources, "\n")
	scan := ParseReviewKeymaps(src)
	if n := len(scan.Resolved) + len(scan.Dynamic) + len(scan.Unresolved); n != strings.Count(src, keymapCall) {
		t.Fatalf("review keymaps without review: desc: documented %d calls of %d", n, strings.Count(src, keymapCall))
	}
	for _, km := range append(append(scan.Resolved, scan.Dynamic...), scan.Unresolved...) {
		if !reviewClassified(km) {
			t.Errorf("review mapping %q (%s) has no help classification", km.Raw, km.Desc)
		}
	}
	sections, err := reviewSections(src)
	if err != nil {
		t.Fatal(err)
	}
	out := Render(Layer(nil, []workbenchshortcut.Chord{workbenchshortcut.ChordAltN}, sections))
	for _, want := range []string{"Alt+a", "Alt+r", "Alt+Shift+A", "Alt+Shift+R", "Alt+q", "Alt+⏎", "Alt+n", "Alt+Shift+N", "]m", "[m", "quote selection", "Esc"} {
		if !strings.Contains(out, want) {
			t.Errorf("review help missing %q:\n%s", want, out)
		}
	}
	if reviewClassified(NvimKeymap{Key: "<M-z>", Raw: "'<M-z>'", Desc: "new mapping"}) {
		t.Fatal("new review binding silently classified")
	}
	// Descriptions are consumed from the live mapping, not copied into Go.
	changed, err := reviewSections(strings.ReplaceAll(src, "review: accept 🤖 suggestion (§5)", "review: changed accept behavior"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(Render(changed), "changed accept behavior") {
		t.Fatal("help wording did not follow mapping desc")
	}
}

// Thread controls live in modules, and q applies only inside the thread float.
// Override their descriptions to prove the composed help reads both sources.
type reviewTreeSources struct{ t *testing.T }

func (s reviewTreeSources) Read(path string) ([]byte, error) {
	text := mustReadTreeSource(s.t, path)
	if path == "nvim/review/comment.lua" {
		text = strings.ReplaceAll(text, "review: open comment thread", "review: open edited thread")
	}
	if path == "nvim/review/comment_float.lua" {
		text = strings.ReplaceAll(text, "review: save and close thread", "review: save edited thread")
	}
	return []byte(text), nil
}
func TestCommentHelpReadsModuleDescriptionsAndScopes(t *testing.T) {
	sections, err := Sections(reviewTreeSources{t})
	if err != nil {
		t.Fatal(err)
	}
	wants := map[string]string{
		"Enter (on comment; normal)": "open edited thread",
		"q (thread float; normal)":   "save edited thread",
	}
	for _, section := range sections {
		for _, binding := range section.Bindings {
			if want, ok := wants[binding.Key]; ok {
				if binding.Desc != want || binding.Context != ContextReview {
					t.Errorf("%s: got %+v, want review scope and %q", binding.Key, binding, want)
				}
				delete(wants, binding.Key)
			}
		}
	}
	for key := range wants {
		t.Errorf("missing scoped help: %s", key)
	}
}

func TestReviewDescriptionsAllowCompactLuaAssignment(t *testing.T) {
	for _, assignment := range []string{"desc='review: thread'", "desc = 'review: thread'", "desc\t=\t'review: thread'"} {
		scan := ParseReviewKeymaps("vim.keymap.set('n','q',action,{" + assignment + "})")
		if len(scan.Resolved) != 1 || scan.Resolved[0].Desc != "thread" {
			t.Errorf("%s: got %+v", assignment, scan)
		}
	}
}
