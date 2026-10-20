package couchcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchscratch"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestScratchHostProbeRequiresPersistentOptIn(t *testing.T) {
	raw, _, _ := scratchWiringRuntime(t)
	rt, lease, err := raw.prepareSingleton(true)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	ns, err := rt.ResolveNamespace()
	if err != nil {
		t.Fatal(err)
	}
	c, err := rt.NewCouchWith(couchcore.NewFakeRunner(), ns)
	if err != nil {
		t.Fatal(err)
	}
	var notes []string
	c.ScratchStartup(t.Context(), func(note string) { notes = append(notes, note) })
	checkout := filepath.Join(rt.isolatedRoot, "fixture")
	scope, err := launcher.ResolveRepoScope(checkout)
	if err != nil {
		t.Fatal(err)
	}
	zero := 0
	c.Slots = &couchcore.SlotCatalogFake{Workspaces: map[string]couchcore.WorkspaceIdentity{checkout: {Repo: "fixture", RepoIdentity: filepath.Join(checkout, ".git"), WorktreeRoot: checkout, PrimaryRoot: checkout, Kind: "primary", Slot: &zero}}}
	record := couchcore.ThreadRecord{SchemaVersion: couchcore.ThreadSchemaVersion, Address: couchcore.ThreadAddress{RepoScope: scope.Key, Tag: "couch-0123456789abcdef"}, StartingPath: checkout, WorkingPath: checkout, CreatedAt: time.Unix(100, 0), Revision: 1}
	host := couchscratch.New(couchscratch.Options{Probe: raw.scratchProbeFn})
	if len(host.PrepareSlot(couchscratch.Slot{Repo: "fixture", Identity: filepath.Join(checkout, ".git"), Number: 0})) != 3 {
		t.Fatal("host write probe must pass")
	}
	marker := couchscratch.ScratchOptInPath(rt.Getenv("HOME"), "")
	notes = nil
	if got := c.SlotEnvironment(t.Context(), record); len(got) != 0 {
		t.Fatalf("undeclared setup exported %v", got)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], marker) {
		t.Fatalf("missing marker note: %v", notes)
	}
	if err := os.MkdirAll(filepath.Dir(marker), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("operator declaration; contents ignored"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := c.SlotEnvironment(t.Context(), record); len(got) != 3 {
		t.Fatalf("declared setup lost overlay: %v", got)
	}
	if got := c.ScratchDiagnostic(t.Context(), record); got != "scratch: RAM disk" {
		t.Fatal(got)
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	if got := c.SlotEnvironment(t.Context(), record); len(got) != 0 {
		t.Fatalf("removed declaration still admitted %v", got)
	}
	if err := os.Mkdir(marker, 0700); err != nil {
		t.Fatal(err)
	}
	if got := c.SlotEnvironment(t.Context(), record); len(got) != 0 {
		t.Fatalf("directory declaration admitted %v", got)
	}
}

func TestShowRendersCurrentScratchAdmission(t *testing.T) {
	var out bytes.Buffer
	render(&out, couchcore.Operation{Name: "show"}, couchcore.ShowResult{Scratch: []string{"tag: scratch: normal temp (opt-in unavailable: /fixture/sandbox-allowed)"}})
	if got := out.String(); !strings.Contains(got, "sandbox-allowed") || !strings.Contains(got, "[current launch selection]") {
		t.Fatal(got)
	}
}

func TestScratchOptInXDGAndIsolation(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(root, "config")
	marker := couchscratch.ScratchOptInPath("", config)
	if err := os.MkdirAll(filepath.Dir(marker), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, nil, 0000); err != nil {
		t.Fatal(err)
	}
	// Contents are never opened: even an unreadable regular declaration works.
	rt := OSRuntime{isolatedRoot: root, env: func(key string) string {
		if key == "XDG_CONFIG_HOME" {
			return config
		}
		return ""
	}, scratchProbeFn: func() couchscratch.Probe {
		return couchscratch.Probe{Root: root, Mounted: true, FreeBytes: couchscratch.Floor}
	}}
	if allowed, status := rt.scratchAdmission(couchcore.ThreadRecord{}); !allowed {
		t.Fatal(status)
	}
	rt.isolatedRoot = filepath.Join(root, "other-isolation")
	if allowed, _ := rt.scratchAdmission(couchcore.ThreadRecord{}); allowed {
		t.Fatal("read ambient opt-in outside isolation")
	}
	rt.isolatedRoot = root
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "absent"), marker); err != nil {
		t.Fatal(err)
	}
	if allowed, _ := rt.scratchAdmission(couchcore.ThreadRecord{}); allowed {
		t.Fatal("symlink declaration admitted")
	}
}
