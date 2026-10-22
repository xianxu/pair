package couchcmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

func TestSessionSyncOperatorForms(t *testing.T) {
	for _, verb := range []string{"operator-acquire", "operator-release"} {
		args := []string{"--session-sync", verb, "document.md", "--id", "0123456789abcdef0123456789abcdef", "--turn", "fedcba9876543210fedcba9876543210", "--json"}
		if _, err := ParseCLI(args, couchcore.Operations()); err != nil {
			t.Fatalf("public operator command refused: %v", err)
		}
		for _, bad := range [][]string{
			{verb, "document.md", "--id", "id", "--turn", "turn"},
			{verb, "document.md", "--turn", "turn", "--json"},
			{verb, "document.md", "--id", "id", "--json"},
			{verb, "document.md", "--id", "id", "--turn", "turn", "--json", "--id", "other"},
			{verb, "document.md", "--id", "id", "--turn", "turn", "--json", "--turn", "other"},
			{verb, "document.md", "--id", "id", "--turn", "turn", "--json", "--json"},
			{verb, "document.md", "--id", "id", "--turn", "turn", "--json", "--owner", "ops:0"},
			{verb, "document.md", "extra", "--id", "id", "--turn", "turn", "--json"},
		} {
			if _, err := ParseCLI(append([]string{"--session-sync"}, bad...), couchcore.Operations()); err == nil {
				t.Fatalf("accepted invalid form %v", bad)
			}
		}
	}
}

const operatorTestTurn = "fedcba9876543210fedcba9876543210"

func operatorCLIRegistration(t *testing.T, rt testRT, file string) string {
	t.Helper()
	code, out, errout := runSS(t, rt, "register", file, "--owner", "ops:0", "--json")
	if code != 0 {
		t.Fatalf("register: exit%d %s %s", code, out, errout)
	}
	var response sessionSyncResponse
	if err := json.Unmarshal([]byte(out), &response); err != nil || response.Registration == nil {
		t.Fatalf("register: %s (%v)", out, err)
	}
	return response.Registration.ID
}

func operatorCLIResponse(t *testing.T, rt testRT, verb, file, id, turn string, exit int) map[string]any {
	t.Helper()
	code, out, errout := runSS(t, rt, verb, file, "--id", id, "--turn", turn, "--json")
	if code != exit || errout != "" {
		t.Fatalf("%s: exit%d want%d stdout=%s stderr=%s", verb, code, exit, out, errout)
	}
	var response map[string]any
	if err := json.Unmarshal([]byte(out), &response); err != nil {
		t.Fatalf("%s: %v", out, err)
	}
	if exit == 0 {
		if response["Code"] != "ok" || response["Error"] != "" {
			t.Fatalf("success envelope: %v", response)
		}
	} else if response["Code"] == "ok" || response["Error"] == "" {
		t.Fatalf("refusal envelope: %v", response)
	}
	return response
}

func TestSessionSyncOperatorLifecycle(t *testing.T) {
	rt, file := sessionSyncRT(t)
	id := operatorCLIRegistration(t, rt, file)
	original, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	first := operatorCLIResponse(t, rt, "operator-acquire", file, id, operatorTestTurn, 0)
	lock, ok := first["OperatorLock"].(map[string]any)
	if !ok {
		t.Fatalf("missing operator payload: %v", first)
	}
	lockPath, _ := lock["LockPath"].(string)
	lockText, _ := lock["LockText"].(string)
	want := map[string]any{"ID": id, "File": file, "Holder": "operator", "Turn": operatorTestTurn, "LockPath": lockPath, "LockText": lockText, "State": "acquired"}
	if lockPath == "" || lockText == "" || !reflect.DeepEqual(lock, want) {
		t.Fatalf("ownership payload: %v", lock)
	}
	if expected := map[string]any{"Code": "ok", "Error": "", "Status": nil, "Registration": nil, "Registrations": nil, "Changed": false, "OperatorLock": want}; !reflect.DeepEqual(first, expected) {
		t.Fatalf("success envelope: %v", first)
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil || string(raw) != lockText {
		t.Fatalf("lock bytes %q, %v", raw, err)
	}
	retry := operatorCLIResponse(t, rt, "operator-acquire", file, id, operatorTestTurn, 0)
	want["State"] = "already-held"
	if !reflect.DeepEqual(retry["OperatorLock"], want) {
		t.Fatalf("retry: %v", retry)
	}
	if after, err := os.ReadFile(lockPath); err != nil || !bytes.Equal(after, raw) {
		t.Fatalf("retry changed proof: %q %v", after, err)
	}
	for _, state := range []string{"released", "already-released"} {
		response := operatorCLIResponse(t, rt, "operator-release", file, id, operatorTestTurn, 0)
		want = map[string]any{"ID": id, "File": file, "Holder": "", "Turn": operatorTestTurn, "LockPath": lockPath, "LockText": "", "State": state}
		if !reflect.DeepEqual(response["OperatorLock"], want) {
			t.Fatalf("release: %v", response)
		}
		if _, err := os.Lstat(lockPath); !os.IsNotExist(err) {
			t.Fatalf("lock remains: %v", err)
		}
	}
	if after, err := os.ReadFile(file); err != nil || !bytes.Equal(after, original) {
		t.Fatalf("ownership changed document: %q %v", after, err)
	}
	if rt.supervisor.acquired != 0 {
		t.Fatal("operator CLI acquired supervisor")
	}
}

func TestSessionSyncOperatorRefusals(t *testing.T) {
	for _, verb := range []string{"operator-acquire", "operator-release"} {
		t.Run(verb, func(t *testing.T) {
			rt, file := sessionSyncRT(t)
			id := operatorCLIRegistration(t, rt, file)
			first := operatorCLIResponse(t, rt, "operator-acquire", file, id, operatorTestTurn, 0)
			lock := first["OperatorLock"].(map[string]any)
			response := operatorCLIResponse(t, rt, verb, file, id, strings.Repeat("1", 32), 3)
			if response["Code"] != "operator-lock" {
				t.Fatalf("foreign ownership: %v", response)
			}
			response = operatorCLIResponse(t, rt, verb, file, strings.Repeat("2", 32), operatorTestTurn, 1)
			if response["Code"] != "operator-identity" {
				t.Fatalf("identity refusal lost: %v", response)
			}
			if raw, err := os.ReadFile(lock["LockPath"].(string)); err != nil || string(raw) != lock["LockText"] {
				t.Fatalf("refusal changed ownership: %q %v", raw, err)
			}
		})
	}
}

func TestSessionSyncOperatorReleaseAbsent(t *testing.T) {
	rt, file := sessionSyncRT(t)
	id := operatorCLIRegistration(t, rt, file)
	response := operatorCLIResponse(t, rt, "operator-release", file, id, operatorTestTurn, 0)
	lock := response["OperatorLock"].(map[string]any)
	if lock["State"] != "already-released" || lock["Holder"] != "" || lock["LockText"] != "" {
		t.Fatalf("absent release claimed ownership: %v", lock)
	}
	if _, err := os.Lstat(sessionsync.PathsOf(file).Lock); !os.IsNotExist(err) {
		t.Fatalf("absent release created lock: %v", err)
	}
}

func TestSessionSyncOperatorStoreUnavailable(t *testing.T) {
	rt, file := sessionSyncRT(t)
	id := operatorCLIRegistration(t, rt, file)
	store := filepath.Join(sessionsync.LedgerDir(rt.namespace.Dir()), "documents")
	if err := os.Chmod(store, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(store, 0700); err != nil {
			t.Error(err)
		}
	})
	if _, err := os.ReadFile(filepath.Join(store, "registry.json")); !os.IsPermission(err) {
		t.Skipf("fixture cannot deny store reads for this user: %v", err)
	}
	for _, verb := range []string{"operator-acquire", "operator-release"} {
		response := operatorCLIResponse(t, rt, verb, file, id, operatorTestTurn, 7)
		if response["Code"] != "session-sync-unavailable" {
			t.Fatalf("typed store unavailability lost: %v", response)
		}
	}
}

type operatorNamespaceFailureRT struct{ testRT }

func (operatorNamespaceFailureRT) ResolveNamespace() (couchcore.CouchNamespace, error) {
	return couchcore.CouchNamespace{}, fmt.Errorf("namespace inaccessible")
}

func TestSessionSyncOperatorNamespaceUnavailable(t *testing.T) {
	rt, file := sessionSyncRT(t)
	for _, verb := range []string{"operator-acquire", "operator-release", "lookup"} {
		args := []string{"--session-sync", verb, file, "--json"}
		if verb != "lookup" {
			args = append(args, "--id", strings.Repeat("1", 32), "--turn", operatorTestTurn)
		}
		var out, errout bytes.Buffer
		code := RunWithRuntime(args, strings.NewReader(""), &out, &errout, operatorNamespaceFailureRT{rt})
		var response map[string]any
		if err := json.Unmarshal(out.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		wantExit, wantCode := 7, "session-sync-unavailable"
		if verb == "lookup" {
			wantExit, wantCode = 1, "error"
		}
		if code != wantExit || response["Code"] != wantCode || response["Error"] == "" || errout.Len() != 0 {
			t.Fatalf("%s: exit%d stdout=%s stderr=%s", verb, code, out.String(), errout.String())
		}
	}
}

func TestSessionSyncOperatorInvalidTurnPrecedesNamespace(t *testing.T) {
	rt, file := sessionSyncRT(t)
	for _, verb := range []string{"operator-acquire", "operator-release"} {
		var out, errout bytes.Buffer
		code := RunWithRuntime([]string{"--session-sync", verb, file, "--id", strings.Repeat("1", 32), "--turn", "invalid", "--json"}, strings.NewReader(""), &out, &errout, operatorNamespaceFailureRT{rt})
		var response map[string]any
		if err := json.Unmarshal(out.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if code != 1 || response["Code"] != "operator-invalid" {
			t.Fatalf("invalid turn became availability fallback: exit%d %s", code, out.String())
		}
	}
}

func TestSessionSyncOperatorUnsafeAndCorruptRefuse(t *testing.T) {
	for _, scenario := range []string{"unsafe-lock", "corrupt-registry", "missing-file"} {
		t.Run(scenario, func(t *testing.T) {
			rt, file := sessionSyncRT(t)
			id := operatorCLIRegistration(t, rt, file)
			switch scenario {
			case "unsafe-lock":
				if err := os.Mkdir(sessionsync.PathsOf(file).Lock, 0700); err != nil {
					t.Fatal(err)
				}
			case "corrupt-registry":
				registry := filepath.Join(sessionsync.LedgerDir(rt.namespace.Dir()), "documents", "registry.json")
				if err := os.WriteFile(registry, []byte("broken"), 0600); err != nil {
					t.Fatal(err)
				}
			case "missing-file":
				if err := os.Remove(file); err != nil {
					t.Fatal(err)
				}
			}
			for _, verb := range []string{"operator-acquire", "operator-release"} {
				response := operatorCLIResponse(t, rt, verb, file, id, operatorTestTurn, 1)
				if response["Code"] == "session-sync-unavailable" {
					t.Fatalf("unsafe failure became unavailable: %v", response)
				}
			}
		})
	}
}
