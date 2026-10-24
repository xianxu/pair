package editorial

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestMigrationMapToTagsPreservesBytesAndItems(t *testing.T) {
	src := []byte("---\nopaque: {keep: yes}\ncollaboration-mode:\n  default: discussion\n  chat: [Notes]\nprocessing-mode: retired\n---\n# Notes (open)\n<!-- presentation-mode: direct -->\n7. ask\n## Child\nbody\n\n# Notes\n3. another\n")
	want := []byte("---\nopaque: {keep: yes}\nprocessing-mode: retired\n---\n@@chat@@\n# Notes (open)\n<!-- presentation-mode: direct -->\n7. ask\n## Child\nbody\n\n@@chat@@\n# Notes\n3. another\n")
	m, err := PlanMigration(src)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(m.After, want) {
		t.Fatalf("got:\n%s\nwant:\n%s", m.After, want)
	}
	digest := sha256.Sum256(src)
	if m.BeforeHash != hex.EncodeToString(digest[:]) {
		t.Fatal("wrong original snapshot hash")
	}
	again, err := PlanMigration(m.After)
	if err != nil || !bytes.Equal(again.After, m.After) {
		t.Fatal("not idempotent", err)
	}
}

func TestMigrationLossRefusesWithSection(t *testing.T) {
	for _, tc := range []struct{ name, src, section string }{
		{"bullet", "---\ncollaboration-mode: {chat: [Notes]}\n---\n# Notes\n- ask\n", "Notes"},
		{"overlapping-tag-bullet", "---\ncollaboration-mode: {chat: [Notes]}\n---\n@@chat@@\n# Notes\n* ask\n", "Notes"},
		{"preamble", "---\ncollaboration-mode: {default: chat}\n---\n1. preamble ask\n# Notes\n2. ask\n", "preamble"},
		{"setext", "---\ncollaboration-mode: {chat: [Notes]}\n---\nNotes\n=====\n1. ask\n", "Notes"},
		{"tag-in-ask-continuation", "---\ncollaboration-mode: {chat: [Notes]}\n---\n# Notes\n1. ask\n# Notes\n2. next\n", "Notes"},
		{"unmatched", "---\ncollaboration-mode: {chat: [Absent]}\n---\n# Present\n", "Absent"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, err := PlanMigration([]byte(tc.src))
			if err == nil || m != nil {
				t.Fatal("lossy migration returned candidate")
			}
			if !strings.Contains(err.Error(), tc.section) {
				t.Fatalf("refusal lacks section %q: %v", tc.section, err)
			}
		})
	}
}

func TestMigrationRetainsIndependentAuthority(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{"comment-bullets", "---\ncollaboration-mode: {chat: [Notes]}\n---\n# Notes\n<!-- collaboration-mode: chat -->\n- ask\n", "---\n---\n# Notes\n<!-- collaboration-mode: chat -->\n- ask\n"},
		{"existing-tag", "---\ncollaboration-mode: {chat: [Notes]}\n---\n@@chat@@\n# Notes\n1. ask\n", "---\n---\n@@chat@@\n# Notes\n1. ask\n"},
		{"default-chat", "---\ncollaboration-mode: {default: chat}\n---\n# Parent\n1. ask\n## Child\n2. next\n\n# Other\n3. last\n", "---\n---\n@@chat@@\n# Parent\n1. ask\n## Child\n2. next\n\n@@chat@@\n# Other\n3. last\n"},
		{"retired-map", "---\ncollaboration-mode: {discussion: [Notes], default: future}\n---\n# Notes\nbody\n", "---\n---\n# Notes\nbody\n"},
		{"scalar-noop", "---\ncollaboration-mode: chat\n---\n- ask\n", "---\ncollaboration-mode: chat\n---\n- ask\n"},
		{"code-heading", "---\ncollaboration-mode: {chat: [Notes]}\n---\n```\n# Notes\n```\n# Notes\n1. ask\n", "---\n---\n```\n# Notes\n```\n@@chat@@\n# Notes\n1. ask\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, err := PlanMigration([]byte(tc.src))
			if err != nil {
				t.Fatal(err)
			}
			if string(m.After) != tc.want {
				t.Fatalf("got:\n%s\nwant:\n%s", m.After, tc.want)
			}
		})
	}
}

func FuzzPlanMigration(f *testing.F) {
	for _, s := range []string{"# A\n", "---\ncollaboration-mode: {chat: [Notes]}\n---\n# Notes\n1. ask\n", "---\ncollaboration-mode: {default: chat}\n---\n- preamble\n# Notes\n1. ask\n"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 65536 {
			t.Skip()
		}
		m, err := PlanMigration([]byte(s))
		if err != nil {
			if m != nil {
				t.Fatal("refusal returned bytes")
			}
			return
		}
		before, err := Compile([]byte(s))
		if err != nil {
			t.Fatal(err)
		}
		after, err := Compile(m.After)
		if err != nil {
			t.Fatal(err)
		}
		if len(before.items) != len(after.items) {
			t.Fatal("eligible ask count changed")
		}
		for i, item := range before.items {
			next := after.items[i]
			if s[item.Range.Start:item.Range.End] != string(m.After[next.Range.Start:next.Range.End]) || item.AppendAt-item.Range.Start != next.AppendAt-next.Range.Start || item.Linked != next.Linked {
				t.Fatal("eligible ask projection changed")
			}
		}
		again, err := PlanMigration(m.After)
		if err != nil || !bytes.Equal(again.After, m.After) {
			t.Fatal("not idempotent", err)
		}
	})
}
