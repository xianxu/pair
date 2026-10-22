package couchcmd

import (
	"encoding/json"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestEditorEndpointCLIClosedForms(t *testing.T) {
	for _, args := range [][]string{
		{"--editor-endpoint", "register", "--json"},
		{"--editor-endpoint", "unregister", "--json"},
		{"--editor-endpoint", "lookup", "--namespace", "/private/store", "--document-id", "0123456789abcdef0123456789abcdef", "--json"},
	} {
		inv, err := ParseCLI(args, couchcore.Operations())
		if err != nil || inv.kind != cliEditorEndpoint {
			t.Fatalf("%q: kind %v, %v", args, inv.kind, err)
		}
	}
	for _, args := range [][]string{
		{"--editor-endpoint"},
		{"--editor-endpoint", "register"},
		{"--editor-endpoint", "register", "--json", "--namespace", "/store"},
		{"--editor-endpoint", "lookup", "--json"},
		{"--editor-endpoint", "lookup", "--namespace", "/store", "--namespace", "/other", "--document-id", "id", "--json"},
		{"--editor-endpoint", "reset", "--json"},
	} {
		if _, err := ParseCLI(args, couchcore.Operations()); err == nil {
			t.Fatalf("accepted %q", args)
		}
	}
}

func TestEditorEndpointDocumentNamespaceProjection(t *testing.T) {
	rt, file := sessionSyncRT(t)
	for _, args := range [][]string{{"register", file, "--owner", "ops:0", "--json"}, {"lookup", file, "--json"}} {
		code, out, stderr := runSS(t, rt, args...)
		if code != 0 {
			t.Fatalf("%q: %d %s", args, code, stderr)
		}
		var reply struct {
			Registration *struct{ Namespace, ID string }
		}
		if err := json.Unmarshal([]byte(out), &reply); err != nil {
			t.Fatal(err)
		}
		if reply.Registration == nil || reply.Registration.Namespace != rt.namespace.Dir() || reply.Registration.ID == "" {
			t.Fatalf("missing namespace binding: %s", out)
		}
	}
}
