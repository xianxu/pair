package editorial

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

func TestPlanMigrationPreservesRouting(t *testing.T) {
	c, _ := DefaultCatalog()
	src := []byte("---\nopaque: {keep: yes}\ncollaboration-mode:\n  default: discussion\n  chat: [Notes]\n---\n# Notes (open)\n<!-- presentation-mode: direct -->\n- ask\n## Child\nbody\n# Notes\nbody\n")
	before, err := Compile(src, c)
	if err != nil {
		t.Fatal(err)
	}
	m, err := PlanMigration(src, c)
	if err != nil {
		t.Fatal(err)
	}
	after, err := Compile(m.After, c)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Headings) != len(after.Headings) {
		t.Fatal("headings changed")
	}
	for i := range before.Regions {
		if before.Regions[i].Effective.Collaboration != after.Regions[i].Effective.Collaboration {
			t.Fatal("routing changed")
		}
	}
	if !bytes.Contains(m.After, []byte("<!-- presentation-mode: direct; collaboration-mode: chat -->")) || !bytes.Contains(m.After, []byte("opaque: {keep: yes}")) {
		t.Fatalf("%s", m.After)
	}
	again, err := PlanMigration(m.After, c)
	if err != nil || !bytes.Equal(again.After, m.After) {
		t.Fatal("not idempotent", err)
	}
}
func TestMigrationUnmatchedRefuses(t *testing.T) {
	c, _ := DefaultCatalog()
	m, e := PlanMigration([]byte("---\ncollaboration-mode: {chat: [Absent]}\n---\n# Present\n"), c)
	if e == nil || m != nil {
		t.Fatal("unmatched migration returned candidate")
	}
}
func TestMigrationAncestorConflictRefuses(t *testing.T) {
	c, _ := DefaultCatalog()
	m, e := PlanMigration([]byte("---\ncollaboration-mode: {chat: [Child]}\n---\n# Parent\n<!-- collaboration-mode: collaboration -->\n## Child\ntext\n"), c)
	if e == nil || m != nil {
		t.Fatal("migration changed inherited comment authority")
	}
}
func FuzzPlanMigration(f *testing.F) {
	for _, s := range []string{"# A\n", "---\ncollaboration-mode: {chat: [Notes]}\n---\n# Notes\nbody\n# Notes (open)\ntext\n", "---\ncollaboration-mode: {default: chat, collaboration: [Child]}\n---\n# Parent\n## Child\n"} {
		f.Add(s)
	}
	c, _ := DefaultCatalog()
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 65536 {
			t.Skip()
		}
		m, e := PlanMigration([]byte(s), c)
		if e != nil {
			if m != nil {
				t.Fatal("refusal returned candidate")
			}
			return
		}
		before, e := Compile([]byte(s), c)
		if e != nil {
			t.Fatal(e)
		}
		after, e := Compile(m.After, c)
		if e != nil {
			t.Fatal(e)
		}
		if len(before.Headings) != len(after.Headings) {
			t.Fatal("heading structure changed")
		}
		for i := range before.Regions {
			a, b := before.Regions[i].Effective, after.Regions[i].Effective
			if a.Collaboration != b.Collaboration || a.Presentation != b.Presentation || a.Processing != b.Processing || a.Instruction != b.Instruction {
				t.Fatal("effective policy changed")
			}
		}
		again, e := PlanMigration(m.After, c)
		if e != nil || !bytes.Equal(m.After, again.After) {
			t.Fatal("migration not idempotent", e)
		}
	})
}
func TestMigrationIndependentRoutingOracle(t *testing.T) {
	c, _ := DefaultCatalog()
	modes := []string{"discussion", "collaboration", "chat"}
	for seed := int64(0); seed < 30; seed++ {
		rng := rand.New(rand.NewSource(seed))
		var src bytes.Buffer
		src.WriteString("---\nopaque: keep exactly\ncollaboration-mode:\n  default: discussion\n")
		assignments := map[string]string{}
		for group, mode := range modes {
			fmt.Fprintf(&src, "  %s: [", mode)
			for i := 0; i < 12; i++ {
				name := fmt.Sprintf("Group%dN%d", group, i)
				if i > 0 {
					src.WriteString(", ")
				}
				src.WriteString(name)
				assignments[strings.ToLower(name)] = mode
			}
			src.WriteString("]\n")
		}
		src.WriteString("---\npreamble\n")
		want := []string{"discussion"}
		type frame struct {
			level int
			mode  string
		}
		stack := []frame{}
		for i := 0; i < 150; i++ {
			level := rng.Intn(6) + 1
			for len(stack) > 0 && stack[len(stack)-1].level >= level {
				stack = stack[:len(stack)-1]
			}
			mode := "discussion"
			if len(stack) > 0 {
				mode = stack[len(stack)-1].mode
			}
			name := fmt.Sprintf("Group%dN%d", rng.Intn(3), rng.Intn(12))
			if i < 36 {
				name = fmt.Sprintf("Group%dN%d", i/12, i%12)
			}
			if i >= 36 && rng.Intn(3) == 0 {
				name = fmt.Sprintf("Unmapped%d", i)
			}
			if mapped, ok := assignments[strings.ToLower(name)]; ok {
				mode = mapped
			}
			fmt.Fprintf(&src, "%s %s (open)\nbody %d\n", strings.Repeat("#", level), name, i)
			want = append(want, mode)
			stack = append(stack, frame{level, mode})
		}
		migration, e := PlanMigration(src.Bytes(), c)
		if e != nil {
			t.Fatal(e)
		}
		after, e := Compile(migration.After, c)
		if e != nil {
			t.Fatal(e)
		}
		for i, mode := range want {
			if after.Regions[i].Effective.Collaboration != mode {
				t.Fatalf("seed %d region %d got %s want %s", seed, i, after.Regions[i].Effective.Collaboration, mode)
			}
		}
		if !bytes.Contains(migration.After, []byte("opaque: keep exactly")) {
			t.Fatal("opaque changed")
		}
	}
}
