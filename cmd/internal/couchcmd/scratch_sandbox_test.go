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

func TestScratchHostProbeCannotAuthorizeSandboxExport(t *testing.T) {
	raw, ram, _ := scratchWiringRuntime(t)
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
	record := couchcore.ThreadRecord{SchemaVersion: couchcore.ThreadSchemaVersion, Address: couchcore.ThreadAddress{RepoScope: scope.Key, Tag: "couch-0123456789abcdef"}, StartingPath: checkout, WorkingPath: checkout, CreatedAt: time.Unix(100, 0), Revision: 1, LatestLaunchProfile: &couchcore.LaunchProfile{Agent: "claude"}}
	settings := filepath.Join(rt.Getenv("HOME"), ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0700); err != nil {
		t.Fatal(err)
	}
	write := func(data string) {
		t.Helper()
		if err := os.WriteFile(settings, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Host write permission is real, and cannot authorize the consumer sandbox.
	host := couchscratch.New(couchscratch.Options{Probe: raw.scratchProbeFn})
	if len(host.PrepareSlot(couchscratch.Slot{Repo: "fixture", Identity: filepath.Join(checkout, ".git"), Number: 0})) != 3 {
		t.Fatal("host probe must pass")
	}
	write(`{"sandbox":{"enabled":true,"filesystem":{"allowWrite":["/tmp"]}}}`)
	notes = nil
	if env := c.SlotEnvironment(t.Context(), record); len(env) != 0 {
		t.Fatalf("host probe incorrectly authorized denied sandbox: %v", env)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "sandbox doesn't allow "+ram) {
		t.Fatalf("missing bounded actionable note: %v", notes)
	}
	if status := c.ScratchDiagnostic(t.Context(), record); !strings.Contains(status, "sandbox doesn't allow "+ram) {
		t.Fatal(status)
	}
	write(`{"sandbox":{"enabled":true,"filesystem":{"allowWrite":["` + ram + `"]}}}`)
	if env := c.SlotEnvironment(t.Context(), record); len(env) != 3 {
		t.Fatalf("allowed sandbox lost RAM: %v", env)
	}
	if status := c.ScratchDiagnostic(t.Context(), record); status != "scratch: RAM disk" {
		t.Fatal(status)
	}
	write(`{`)
	if env := c.SlotEnvironment(t.Context(), record); len(env) != 0 {
		t.Fatalf("malformed settings authorized RAM: %v", env)
	}
}

func TestShowRendersCurrentScratchAdmission(t *testing.T) {
	var out bytes.Buffer
	render(&out, couchcore.Operation{Name: "show"}, couchcore.ShowResult{Scratch: []string{"tag: scratch: normal temp (sandbox doesn't allow /Volumes/TestScratch)"}})
	if got := out.String(); !strings.Contains(got, "scratch: normal temp") || !strings.Contains(got, "[current launch selection]") {
		t.Fatal(got)
	}
}
