package editorial

import (
	"strings"
	"testing"
)

func TestCompileSharedChatContract(t *testing.T) {
	catalog, err := DefaultCatalog()
	if err != nil {
		t.Fatal(err)
	}
	source := "# Ordinary\nA paragraph.\n\n@@chat@@\n## Questions\n1. Why wait?\n\n# Ordinary again\nAnother paragraph.\n"
	policy, err := Compile([]byte(source), catalog)
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		text string
		chat bool
	}{
		{"A paragraph.", false}, {"1. Why wait?", true}, {"Another paragraph.", false},
	} {
		at := strings.Index(source, check.text)
		found := false
		for _, region := range policy.Regions {
			if region.Range.Start <= at && at < region.Range.End {
				found = true
				if got := region.Effective.Collaboration == "chat"; got != check.chat {
					t.Errorf("%q chat = %v, want %v", check.text, got, check.chat)
				}
			}
		}
		if !found {
			t.Errorf("no policy region for %q", check.text)
		}
	}
}

func TestCompileRetiredValuesNeverRefuse(t *testing.T) {
	catalog, err := DefaultCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, metadata := range []string{
		"presentation-mode: direct\nprocessing-mode: no-such-mode\n",
		"presentation-mode: [unterminated\n",
		"collaboration-mode: unknown\n",
		"processing-mode: one\nprocessing-mode: two\n",
		"process-instruction: '[missing] arbitrary text'\n",
	} {
		t.Run(metadata, func(t *testing.T) {
			source := "---\n" + metadata + "---\n# Text\nOperator content.\n"
			policy, err := Compile([]byte(source), catalog)
			if err != nil {
				t.Fatalf("retired metadata must be nonfatal: %v", err)
			}
			if policy.Defaults.Presentation != "highlight" {
				t.Errorf("new edits use %q, want highlight", policy.Defaults.Presentation)
			}
			if len(policy.Diagnostics) == 0 {
				t.Error("legacy metadata needs a compatibility notice")
			}
		})
	}
}

func TestCollaborativeTagsAreOwnedUnderWholeLegacyChat(t *testing.T) {
	catalog, err := DefaultCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, prefix := range []string{"", "---\ncollaboration-mode: chat\n---\n"} {
		t.Run(prefix, func(t *testing.T) {
			source := prefix + "@@chat@@\n# Questions\n1. Why?\n"
			policy, err := Compile([]byte(source), catalog)
			if err != nil {
				t.Fatal(err)
			}
			at := strings.Index(source, "@@chat@@")
			_, err = ValidateEdits([]byte(source), policy, []Change{{Start: at, End: at + len("@@chat@@\n")}}, catalog)
			refusal, ok := err.(*PolicyError)
			if !ok || refusal.Code != "operator-owned" {
				t.Fatalf("deleting a real tag must refuse operator-owned, got %v", err)
			}
		})
	}
}
