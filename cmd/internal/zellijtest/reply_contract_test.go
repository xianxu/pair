package zellijtest_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

type fixtureReplyCase struct {
	Name, Source, Kind, Expect, Decoder string
	RequestedName                       string `json:"requested_name"`
	Payload                             json.RawMessage
	Reply                               zellijtest.ReplyEnvelope
}

func fixtureRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no repository root")
		}
		dir = parent
	}
}

func TestFixtureReplyGeneratorUnitContracts(t *testing.T) {
	path := filepath.Join(fixtureRoot(t), "tests/lib/zellij_fixture_replies_test.py")
	if raw, err := exec.Command("python3", "-B", path).CombinedOutput(); err != nil {
		t.Fatalf("generator unit contracts: %v: %s", err, raw)
	}
}

// TestEveryZellijFixtureReplyUsesProductionDecoder enumerates producer-specific
// seeds through the same serializer used by the fake, including explicit faults.
// It is deliberately independent of any positive test's branch being exercised.
func TestEveryZellijFixtureReplyUsesProductionDecoder(t *testing.T) {
	root := fixtureRoot(t)
	raw, err := exec.Command("python3", "-B", filepath.Join(root, "tests/lib/zellij_fixture_replies.py"), "--catalog").CombinedOutput()
	if err != nil {
		t.Fatalf("catalog: %v: %s", err, raw)
	}
	var cases []fixtureReplyCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("no fixture reply cases registered")
	}
	sources := map[string]bool{}
	for _, c := range cases {
		source, function, _ := strings.Cut(c.Source, ":")
		sources[source] = true
		t.Run(c.Name, func(t *testing.T) {
			if c.Source == "" {
				t.Fatal("case has no producer")
			}
			body, err := os.ReadFile(filepath.Join(root, source))
			if err != nil {
				t.Fatal(err)
			}
			if function != "" && !strings.Contains(string(body), function) {
				t.Fatal("declared producer function is missing")
			}
			if c.Kind == "fault" && c.Decoder == "" {
				t.Fatal("fault requires an explicit decoder classification")
			}
			kind := c.Kind
			if kind == "fault" {
				kind = c.Decoder
			}
			if !map[string]bool{"panes": true, "clients": true, "tab": true, "visibility": true, "sessions-short": true, "sessions-full": true, "no-sessions": true, "no-session": true, "not-found": true, "name-invalid": true, "name": true}[kind] {
				t.Fatalf("unclassified decoder %q", kind)
			}
			if c.Expect != "accept" && c.Expect != "reject" {
				t.Fatal("explicit accept/reject required")
			}
			err = decodeFixtureReply(t, c)
			if (err == nil) != (c.Expect == "accept") {
				t.Fatalf("%s/%s: decode=%v; envelope=%+v", c.Source, c.Kind, err, c.Reply)
			}
		})
	}
	for _, source := range []string{
		"tests/lib/fake-zellij.py", "tests/term-pane-shortcuts-test.sh", "tests/copy-on-select-test.sh",
		"tests/review-window-test.sh", "tests/review-controls-test.sh", "tests/review-poke-test.sh",
		"tests/overlay-return-test.sh", "tests/review-toggle-test.sh", "tests/review-fresh-restore-test.sh",
		"tests/review-branch-restore-test.sh", "tests/workbench-route-nvim-test.sh",
		"tests/pair-embedded-runtime-test.sh", "tests/review-comments-test.sh",
		"cmd/pair-go/main_test.go", "cmd/internal/pairlifecycletest/zellij_stub.go",
		"cmd/internal/couchcore/terminal_registration_fixture_test.go", "cmd/internal/couchcore/park_client_readiness_test.go",
		"cmd/internal/couchcmd/owned_query_test.go", "cmd/internal/wrapcmd/translate_test.go",
	} {
		if !sources[source] {
			t.Errorf("CLI reply producer lacks a conformance case: %s", source)
		}
	}
	checkFixtureProducerInventory(t, root, sources)
}

// Discovery makes a newly added subprocess fake a visible contract obligation.
// These exclusions produce no fake native query reply; keep their reasons here.
func checkFixtureProducerInventory(t *testing.T, root string, sources map[string]bool) {
	t.Helper()
	excluded := map[string]string{
		"cmd/internal/zellijtest/reply_contract_test.go":            "contract runner itself writes the shared envelope",
		"cmd/internal/couchcore/startup_detach_timing_live_test.go": "optional native passthrough captures real installed replies verbatim",
		"cmd/internal/couchcore/recovery_conformance_live_test.go":  "optional native fixture queries installed server; shell is an agent publisher",
		"cmd/internal/couchcore/recovery_query_capture_test.go":     "cancellation wrapper publishes file readiness then execs the shared stateful fake; it emits no query reply",
		"cmd/internal/clipcmd/admission_test.go":                    "shell emits only mutation failure; queries use shared stateful fake",
		"cmd/internal/layoutcmd/terminal_recovery_live_test.go":     "optional native fixture queries the installed server; shell is only a pane-input receiver and emits no query reply",
	}
	for _, tree := range []string{"cmd", "tests"} {
		err := filepath.WalkDir(filepath.Join(root, tree), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if strings.Contains(path, "runtimebundle/assets") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, "_test.go") && !strings.HasSuffix(path, ".sh") && !strings.HasSuffix(path, ".py") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			body := string(raw)
			query := false
			for _, q := range []string{"list-clients", "list-panes", "current-tab-info", "list-sessions"} {
				query = query || strings.Contains(body, q)
			}
			stub := false
			for _, s := range []string{"#!/bin/sh", "#!/usr/bin/env python", "#!/bin/bash", "#!/usr/bin/env bash"} {
				stub = stub || strings.Contains(body, s)
			}
			if !query || !stub {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			if _, ok := excluded[rel]; ok {
				return nil
			}
			if !sources[rel] {
				t.Errorf("unregistered CLI reply producer %s: add producer-specific generated cases", rel)
				return nil
			}
			shared := false
			for _, marker := range []string{"zellij_fixture_replies", "zellijtest.Case(", "zellijtest.ShellCase(", "zellijtest.Reply(", "zellijtest.ShellReply(", "zellijtest.ReplyCommand("} {
				shared = shared || strings.Contains(body, marker)
			}
			if !shared {
				t.Errorf("%s is registered but does not use the shared reply generator", rel)
			}
			for _, legacy := range []string{"printf 'CLIENT_ID", "printf '[{", "print(json.dumps("} {
				if strings.Contains(body, legacy) {
					t.Errorf("%s still independently serializes a native reply (%s); use the shared generator, or an explicit generated fault", rel, legacy)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func decodeFixtureReply(t *testing.T, c fixtureReplyCase) error {
	t.Helper()
	kind := c.Kind
	if kind == "fault" {
		kind = c.Decoder
		if kind == "" {
			return fmt.Errorf("fault requires explicit decoder")
		}
	}
	r := c.Reply
	switch kind {
	case "panes":
		if r.Code != 0 {
			return fmt.Errorf("exit %d", r.Code)
		}
		_, err := zellijreply.ParsePanes([]byte(r.Stdout), zellijreply.PaneRequirements{Command: true, ActualCommand: true, State: true, Geometry: true, Tab: true})
		return err
	case "clients":
		if r.Code != 0 {
			return fmt.Errorf("exit %d", r.Code)
		}
		_, err := zellijreply.ParseClients([]byte(r.Stdout))
		return err
	case "tab":
		if r.Code != 0 {
			return fmt.Errorf("exit %d", r.Code)
		}
		tab, err := zellijreply.ParseTab([]byte(r.Stdout))
		if err == nil && (tab.ID == nil || tab.Position == nil) {
			return fmt.Errorf("missing correlated tab identity")
		}
		return err
	case "visibility":
		if (r.Code == 0 && r.Stdout == "true\n" && r.Stderr == "") || (r.Code == 1 && r.Stdout == "false\n" && r.Stderr == "") {
			return nil
		}
		return fmt.Errorf("invalid visibility envelope")
	case "no-session", "not-found", "name-invalid", "name":
		name := "pair-fixture"
		var p struct{ Name string }
		_ = json.Unmarshal(c.Payload, &p)
		if p.Name != "" {
			name = p.Name
		}
		if c.RequestedName != "" {
			name = c.RequestedName
		}
		bin := t.TempDir()
		path := filepath.Join(bin, "zellij")
		body := "#!/bin/sh\n" + r.Shell() + "\n"
		if err := os.WriteFile(path, []byte(body), 0700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		err := (launcher.OSRuntime{}).ProbeSessionNameContext(context.Background(), name)
		if kind == "name-invalid" && err != nil && strings.Contains(err.Error(), "session name too long") {
			return nil
		}
		return err
	case "sessions-short", "sessions-full", "no-sessions":
		bin := t.TempDir()
		path := filepath.Join(bin, "zellij")
		short, full := r, r
		if c.Kind != "fault" && kind != "no-sessions" {
			var payload any
			if err := json.Unmarshal(c.Payload, &payload); err != nil {
				return err
			}
			short = zellijtest.Reply(t, "sessions-short", payload)
			full = zellijtest.Reply(t, "sessions-full", payload)
			if kind == "sessions-short" {
				short = r
			} else {
				full = r
			}
		}
		body := "#!/bin/sh\ncase \"$*\" in 'list-sessions --short') " + short.Shell() + " ;; 'list-sessions --no-formatting') " + full.Shell() + " ;; *) exit 97 ;; esac\n"
		if err := os.WriteFile(path, []byte(body), 0700); err != nil {
			t.Fatal(err)
		}
		_, err := (launcher.ZellijSource{Path: path}).LivenessContext(context.Background())
		return err
	default:
		return fmt.Errorf("unclassified reply kind %q", kind)
	}
}
