package sessionsync

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFrontmatterBytes(t *testing.T) {
	for _, nl := range []string{"\n", "\r\n"} {
		block := strings.ReplaceAll("---\nowner: ops:0\n---", "\n", nl)
		tail := nl + "\nUpdated 2026-10-10 23:43\n## A\nbody\x00\n---\n"
		got, err := FrontmatterBody(block + tail)
		if err != nil || got != block {
			t.Fatalf("block %q: %q %v", block, got, err)
		}
		for _, suffix := range []string{"", "\n", "\r\n\n"} {
			replacement := "---\nowner: ops:0\nindex: new\n---"
			next, err := ReplaceFrontmatter(block+tail, replacement+suffix)
			if err != nil || next != replacement+tail {
				t.Fatalf("replace %q: %q %v", suffix, next, err)
			}
		}
	}
	for _, bad := range []string{"", "## Body\n---\n", "---\nowner: ops:0\n"} {
		if _, err := FrontmatterBody(bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	if _, err := ReplaceFrontmatter("---\n---\nbody", "---\n---\nsmuggled body"); err == nil {
		t.Fatal("accepted body payload")
	}
}

func FuzzFrontmatterReplacement(f *testing.F) {
	f.Add([]byte("value"), "\r\nbody\x00", true)
	f.Add([]byte{}, "", false)
	f.Fuzz(func(t *testing.T, value []byte, tail string, crlf bool) {
		nl := "\n"
		if crlf {
			nl = "\r\n"
		}
		block := "---" + nl + "value: " + hex.EncodeToString(value) + nl + "---"
		original := block + nl + tail
		got, err := FrontmatterBody(original)
		if err != nil || got != block {
			t.Fatalf("projection %q %v", got, err)
		}
		replacement := "---\nvalue: next\n---"
		next, err := ReplaceFrontmatter(original, replacement+"\n")
		if err != nil || next != replacement+nl+tail {
			t.Fatalf("suffix lost: %q %v", next, err)
		}
		roundtrip, err := ReplaceFrontmatter(next, got)
		if err != nil || roundtrip != original {
			t.Fatalf("roundtrip %q %v", roundtrip, err)
		}
		// Arbitrary malformed input must either refuse or identify a prefix.
		if prefix, err := FrontmatterBody(string(value)); err == nil && !strings.HasPrefix(string(value), prefix) {
			t.Fatal("not a prefix")
		}
	})
}

func TestFrontmatterPreservesUnreadBody(t *testing.T) {
	for _, registered := range []bool{false, true} {
		for _, baseline := range []bool{false, true} {
			t.Run(strings.Join([]string{map[bool]string{true: "registered", false: "legacy"}[registered], map[bool]string{true: "baseline", false: "first"}[baseline]}, "/"), func(t *testing.T) {
				r := newRig(t)
				if !baseline {
					if err := os.Remove(r.p.Shadow); err != nil {
						t.Fatal(err)
					}
				}
				if registered {
					if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
						t.Fatal(err)
					}
				}
				edited := strings.Replace(statusDoc, "old", "operator unread", 1)
				r.put(r.p.File, edited)
				beforeBody := edited[strings.Index(edited, "---\n\n")+3:]
				read, err := r.sync.ReadFrontmatter(r.p.File)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(r.diffText(), "+operator unread") || strings.Contains(read.Body, "##") {
					t.Fatalf("read consumed body: %q", r.diffText())
				}
				block := strings.Replace(read.Body, "owner: ops:0", "owner: ops:0\ndiscussions: [new]", 1)
				res, err := r.sync.Write(WriteRequest{File: r.p.File, Frontmatter: true, Body: block + "\n", IfHash: read.Hash})
				if err != nil {
					t.Fatal(err)
				}
				if r.read(r.p.File) != block+beforeBody || res.Hash != Hash(block) {
					t.Fatalf("body changed: %q", r.read(r.p.File))
				}
				diff := r.diffText()
				if !strings.Contains(diff, "+operator unread") || strings.Contains(diff, "+discussions:") {
					t.Fatalf("wrong baseline: %q", diff)
				}
			})
		}
	}
}

func TestFrontmatterGuardsAndHorizon(t *testing.T) {
	r := newRig(t)
	r.deliver("d1", "report")
	read, err := r.sync.ReadFrontmatter(r.p.File)
	if err != nil {
		t.Fatal(err)
	}
	if ids := r.owedIDs(); len(ids) != 1 {
		t.Fatalf("read lost owed rows: %v", ids)
	}
	for _, mode := range []string{"stale", "lock", "owner", "payload", "scope"} {
		t.Run(mode, func(t *testing.T) {
			req := WriteRequest{File: r.p.File, Frontmatter: true, Body: read.Body, IfHash: read.Hash}
			r.put(r.p.File, statusDoc)
			switch mode {
			case "stale":
				r.put(r.p.File, strings.Replace(statusDoc, "owner: ops:0", "owner: ops:0\nother: true", 1))
			case "lock":
				r.put(r.p.Lock, "holder: operator\n")
				defer os.Remove(r.p.Lock)
			case "owner":
				req.Body = strings.Replace(read.Body, "ops:0", "ops:1", 1)
			case "payload":
				req.Body += "\n## Other\n"
			case "scope":
				req.Section = "Needs you"
			}
			before := []string{r.read(r.p.File), r.read(r.p.Shadow), r.read(r.p.State)}
			_, err := r.sync.Write(req)
			if err == nil {
				t.Fatal("guard did not refuse")
			}
			if mode == "stale" {
				wantRefusal(t, err, ExitStale, "stale")
			}
			if mode == "lock" {
				wantRefusal(t, err, ExitOperatorLock, "operator-lock")
			}
			after := []string{r.read(r.p.File), r.read(r.p.Shadow), r.read(r.p.State)}
			for i := range before {
				if before[i] != after[i] {
					t.Fatalf("guard changed file %d", i)
				}
			}
		})
	}
}

func TestFrontmatterPublicationFailureIsConservative(t *testing.T) {
	r := newRig(t)
	r.deliver("d1", "report")
	r.put(r.p.File, strings.Replace(statusDoc, "old", "unread", 1))
	f, err := load(r.p.File)
	if err != nil {
		t.Fatal(err)
	}
	oldShadow := r.read(r.p.Shadow)
	f.paths.Shadow = filepath.Join(t.TempDir(), "directory")
	if err := os.Mkdir(f.paths.Shadow, 0700); err != nil {
		t.Fatal(err)
	}
	block, err := FrontmatterBody(f.content)
	if err != nil {
		t.Fatal(err)
	}
	next := strings.Replace(block, "owner: ops:0", "owner: ops:0\ndiscussions: [new]", 1)
	_, err = r.sync.write(f, WriteRequest{File: r.p.File, Frontmatter: true, Body: next, IfHash: Hash(block), Fold: []string{"d1"}})
	if err == nil {
		t.Fatal("shadow publication unexpectedly succeeded")
	}
	if r.read(r.p.Shadow) != oldShadow || !strings.Contains(r.read(r.p.File), "unread") || !strings.Contains(r.read(r.p.File), "discussions: [new]") {
		t.Fatal("partial result lost body or baseline")
	}
	if ids := r.owedIDs(); len(ids) != 1 {
		t.Fatalf("failure lost owed events: %v", ids)
	}
	read, err := r.sync.ReadFrontmatter(r.p.File)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Frontmatter: true, Body: read.Body, IfHash: read.Hash, Fold: []string{"d1"}}); err != nil {
		t.Fatal(err)
	}
	if ids := r.owedIDs(); len(ids) != 0 || !strings.Contains(r.diffText(), "+unread") {
		t.Fatalf("retry %v %q", ids, r.diffText())
	}
}

func TestFrontmatterFirstWriteDoesNotAcknowledgeBody(t *testing.T) {
	for _, registered := range []bool{false, true} {
		r := newRig(t)
		if err := os.Remove(r.p.Shadow); err != nil {
			t.Fatal(err)
		}
		if registered {
			if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
				t.Fatal(err)
			}
		}
		block, err := FrontmatterBody(statusDoc)
		if err != nil {
			t.Fatal(err)
		}
		r.put(r.p.File, strings.Replace(statusDoc, "old", "unread", 1))
		next := strings.Replace(block, "owner: ops:0", "owner: ops:0\ndiscussions: []", 1)
		if _, err := r.sync.Write(WriteRequest{File: r.p.File, Frontmatter: true, Body: next, IfHash: Hash(block)}); err != nil {
			t.Fatal(err)
		}
		if diff := r.diffText(); !strings.Contains(diff, "+unread") || strings.Contains(diff, "+discussions:") {
			t.Fatalf("registered=%v diff %q", registered, diff)
		}
	}
}

func TestFrontmatterProjectionOntoOrdinaryRegisteredBaseline(t *testing.T) {
	r := newRig(t)
	original := "## Work\n\nold\n"
	r.put(r.p.File, original)
	r.put(r.p.Shadow, original)
	if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
		t.Fatal(err)
	}
	// Registration of ordinary documents starts without a legacy shadow.
	if _, err := r.sync.Read(r.p.File, ""); err != nil {
		t.Fatal(err)
	}
	block := "---\ndiscussions: [operator]\n---"
	r.put(r.p.File, block+"\n"+strings.Replace(original, "old", "unread", 1))
	read, err := r.sync.ReadFrontmatter(r.p.File)
	if err != nil {
		t.Fatal(err)
	}
	next := strings.Replace(block, "operator", "agent", 1)
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Frontmatter: true, Body: next, IfHash: read.Hash}); err != nil {
		t.Fatal(err)
	}
	if diff := r.diffText(); !strings.Contains(diff, "-old") || !strings.Contains(diff, "+unread") || strings.Contains(diff, "discussions:") {
		t.Fatalf("ordinary baseline lost: %q", diff)
	}
}

func TestFrontmatterMalformedShadowRefusesBeforeEffects(t *testing.T) {
	for _, write := range []bool{false, true} {
		r := newRig(t)
		r.put(r.p.Shadow, "---\nunterminated baseline\n## Old\n")
		before := []string{r.read(r.p.File), r.read(r.p.Shadow), r.read(r.p.State)}
		var err error
		if write {
			block, _ := FrontmatterBody(statusDoc)
			_, err = r.sync.Write(WriteRequest{File: r.p.File, Frontmatter: true, Body: strings.Replace(block, "owner: ops:0", "owner: ops:0\nx: y", 1), IfHash: Hash(block)})
		} else {
			_, err = r.sync.ReadFrontmatter(r.p.File)
		}
		if err == nil {
			t.Fatal("malformed baseline accepted")
		}
		after := []string{r.read(r.p.File), r.read(r.p.Shadow), r.read(r.p.State)}
		for i := range before {
			if before[i] != after[i] {
				t.Fatalf("file %d changed", i)
			}
		}
	}
}
