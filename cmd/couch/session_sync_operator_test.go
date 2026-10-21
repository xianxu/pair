package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

type operatorExecutable struct {
	bin, root, store string
	env              []string
}

func newOperatorExecutable(t *testing.T) operatorExecutable {
	t.Helper()
	bin, _ := installedExecutables(t)
	f := operatorExecutable{bin: bin, env: installedEnv(t, filepath.Dir(bin))}
	for _, entry := range f.env {
		if value, ok := strings.CutPrefix(entry, "COUCH_ISOLATED_ROOT="); ok {
			f.root = value
		}
		if value, ok := strings.CutPrefix(entry, "COUCH_STORE_DIR="); ok {
			f.store = value
		}
	}
	if f.root == "" || f.store == "" {
		t.Fatal("installed environment omitted private roots")
	}
	return f
}

func (f operatorExecutable) run(t *testing.T, discard bool, wantExit int, args ...string) map[string]any {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), f.bin, append([]string{"--session-sync"}, args...)...)
	cmd.Dir, cmd.Env = f.root, f.env
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if discard {
		cmd.Stdout = io.Discard
	}
	err := cmd.Run()
	exit := 0
	if err != nil {
		var failed *exec.ExitError
		if !errors.As(err, &failed) {
			t.Fatalf("%v: %v", args, err)
		}
		exit = failed.ExitCode()
	}
	if exit != wantExit || stderr.Len() != 0 {
		t.Fatalf("%v: exit%d want%d stdout=%s stderr=%s", args, exit, wantExit, out.String(), stderr.String())
	}
	if discard {
		return nil
	}
	var response map[string]any
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatalf("%v: invalid JSON %q: %v", args, out.String(), err)
	}
	if wantExit == 0 {
		if response["Code"] != "ok" || response["Error"] != "" {
			t.Fatalf("success envelope: %v", response)
		}
	} else if response["Code"] == "ok" || response["Error"] == "" {
		t.Fatalf("refusal envelope: %v", response)
	}
	return response
}

func (f operatorExecutable) operator(t *testing.T, discard bool, exit int, verb, file, id, turn string) map[string]any {
	t.Helper()
	return f.run(t, discard, exit, verb, file, "--id", id, "--turn", turn, "--json")
}

func (f operatorExecutable) adopt(t *testing.T) {
	t.Helper()
	args := []string{"--adopt-store", f.store}
	for attempt := 0; attempt < 2; attempt++ {
		cmd := exec.CommandContext(t.Context(), f.bin, args...)
		cmd.Dir, cmd.Env = f.root, f.env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("private namespace adoption: %v %s", err, out)
		}
		var response map[string]any
		if err := json.Unmarshal(out, &response); err != nil {
			t.Fatal(err)
		}
		if attempt == 0 {
			digest, ok := response["digest"].(string)
			if !ok || digest == "" || response["status"] != "READY" {
				t.Fatalf("adoption preview: %v", response)
			}
			args = append(args, "--apply", digest)
		} else if response["status"] != "SELECTED" {
			t.Fatalf("adoption: %v", response)
		}
	}
}

func TestSessionSyncOperatorExecutableUnavailable(t *testing.T) {
	for _, verb := range []string{"operator-acquire", "operator-release"} {
		for _, scenario := range []string{"invalid-private-root", "unadopted-namespace"} {
			t.Run(verb+"/"+scenario, func(t *testing.T) {
				f := newOperatorExecutable(t)
				// A private root setting outside the declared isolation is invalid;
				// resolving it must not fall through to the user's live namespace.
				outside := t.TempDir()
				if scenario == "invalid-private-root" {
					for i, entry := range f.env {
						if strings.HasPrefix(entry, "COUCH_STORE_DIR=") {
							f.env[i] = "COUCH_STORE_DIR=" + outside
						}
					}
				}
				response := f.operator(t, false, 7, verb, filepath.Join(f.root, "document.md"), strings.Repeat("1", 32), strings.Repeat("2", 32))
				if response["Code"] != "session-sync-unavailable" {
					t.Fatalf("unavailable namespace: %v", response)
				}
				entries, err := os.ReadDir(outside)
				if err != nil || len(entries) != 0 {
					t.Fatalf("invalid namespace performed effects: %v %v", entries, err)
				}
			})
		}
	}
}

func TestSessionSyncOperatorExecutableLifecycle(t *testing.T) {
	f := newOperatorExecutable(t)
	f.adopt(t)
	namespace, err := couchcore.ExistingCouchNamespace(f.store)
	if err != nil {
		t.Fatal(err)
	}
	assertStopped := func() {
		t.Helper()
		observation, err := couchcore.ObserveSupervisor(namespace, couchcore.OSProcOps{}, nil)
		if err != nil || observation.Held {
			t.Fatalf("private daemon is not stopped: %+v %v", observation, err)
		}
	}
	assertStopped()
	file := filepath.Join(f.root, "document.md")
	doc := []byte("## Work\n\noriginal\n")
	if err := os.WriteFile(file, doc, 0600); err != nil {
		t.Fatal(err)
	}
	registered := f.run(t, false, 0, "register", file, "--owner", "ops:0", "--json")
	id := registered["Registration"].(map[string]any)["ID"].(string)
	turn := strings.Repeat("a", 32)
	first := f.operator(t, false, 0, "operator-acquire", file, id, turn)["OperatorLock"].(map[string]any)
	lockPath, lockText := file+".lock", first["LockText"].(string)
	want := map[string]any{"ID": id, "File": file, "Holder": "operator", "Turn": turn, "LockPath": lockPath, "LockText": lockText, "State": "acquired"}
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("acquire payload: %v", first)
	}
	var proofTime string
	for _, line := range strings.Split(lockText, "\n") {
		if value, ok := strings.CutPrefix(line, "time: "); ok {
			proofTime = value
		}
	}
	if _, err := time.Parse(time.RFC3339Nano, proofTime); err != nil {
		t.Fatalf("proof time: %q %v", proofTime, err)
	}
	// Model a completed operation whose reply the editor did not receive.
	f.operator(t, true, 0, "operator-acquire", file, id, turn)
	retry := f.operator(t, false, 0, "operator-acquire", file, id, turn)["OperatorLock"]
	want["State"] = "already-held"
	if !reflect.DeepEqual(retry, want) {
		t.Fatalf("lost reply changed proof or time: %v", retry)
	}
	if raw, err := os.ReadFile(lockPath); err != nil || string(raw) != lockText {
		t.Fatalf("persisted proof differs: %q %v", raw, err)
	}
	// Older move/alias ownership can exist only at an admitted alias. The
	// editor still requires the canonical on-disk acknowledgment witness.
	alias := filepath.Join(f.root, "document-alias.md")
	if err := os.Symlink(file, alias); err != nil {
		t.Fatal(err)
	}
	f.run(t, false, 0, "register", alias, "--owner", "ops:0", "--json")
	if err := os.Rename(lockPath, alias+".lock"); err != nil {
		t.Fatal(err)
	}
	canonicalized := f.operator(t, false, 0, "operator-acquire", alias, id, turn)["OperatorLock"]
	if !reflect.DeepEqual(canonicalized, want) {
		t.Fatalf("alias recovery did not normalize canonical witness: %v", canonicalized)
	}
	if raw, err := os.ReadFile(lockPath); err != nil || string(raw) != lockText {
		t.Fatalf("canonical witness not present: %q %v", raw, err)
	}

	// Read before the no-mutation snapshot: read acknowledges the baseline.
	read := f.run(t, false, 0, "read", "--all", file, "--json")
	hash := read["Read"].(map[string]any)["Hash"].(string)
	body := filepath.Join(f.root, "body.txt")
	if err := os.WriteFile(body, []byte("replacement\n"), 0600); err != nil {
		t.Fatal(err)
	}
	paths := sessionsync.PathsOf(file)
	central := filepath.Join(sessionsync.LedgerDir(f.store), "documents", "state", id)
	metadata := []string{file, paths.Lock, paths.Shadow, paths.State, paths.Round, filepath.Join(central, "shadow"), filepath.Join(central, "state.json"), filepath.Join(central, "round.json"), filepath.Join(sessionsync.LedgerDir(f.store), "documents", "registry.json")}
	type value struct {
		Bytes    string
		Exists   bool
		Mode     os.FileMode
		Modified time.Time
	}
	snapshot := func() map[string]value {
		t.Helper()
		out := map[string]value{}
		for _, path := range metadata {
			raw, err := os.ReadFile(path)
			if errors.Is(err, os.ErrNotExist) {
				out[path] = value{}
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			out[path] = value{string(raw), true, info.Mode(), info.ModTime()}
		}
		return out
	}
	before := snapshot()
	// The write verb's existing non-JSON refusal must still report exit 3.
	cmd := exec.CommandContext(t.Context(), f.bin, "--session-sync", "write", "--all", body, file, "--if-hash", hash)
	cmd.Dir, cmd.Env = f.root, f.env
	out, err := cmd.CombinedOutput()
	var failed *exec.ExitError
	if !errors.As(err, &failed) || failed.ExitCode() != 3 || !strings.Contains(string(out), "operator holds the lock") {
		t.Fatalf("protected write: %v %s", err, out)
	}
	if after := snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("refused write changed document/metadata: before=%+v after=%+v", before, after)
	}
	if response := f.operator(t, false, 1, "operator-acquire", file, strings.Repeat("b", 32), turn); response["Code"] != "operator-identity" {
		t.Fatalf("identity: %v", response)
	}
	if response := f.operator(t, false, 1, "operator-acquire", body, id, turn); response["Code"] != "operator-path" {
		t.Fatalf("unregistered path: %v", response)
	}
	if response := f.operator(t, false, 3, "operator-release", file, id, strings.Repeat("c", 32)); response["Code"] != "operator-lock" {
		t.Fatalf("foreign ownership: %v", response)
	}

	// The move journal preserves the raw proof before registry rename.
	newFile := filepath.Join(f.root, "renamed.md")
	if err := os.WriteFile(newFile+".lock", []byte(lockText), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(file, newFile); err != nil {
		t.Fatal(err)
	}
	moved := f.run(t, false, 0, "rename", file, newFile, "--json")
	if moved["Registration"].(map[string]any)["ID"] != id {
		t.Fatalf("move identity: %v", moved)
	}
	persisted, err := os.ReadFile(newFile + ".lock")
	if err != nil || string(persisted) != lockText {
		t.Fatalf("move proof: %q %v", persisted, err)
	}
	recovered := ""
	for _, line := range strings.Split(string(persisted), "\n") {
		if value, ok := strings.CutPrefix(line, "turn: "); ok {
			recovered = value
		}
	}
	if recovered != turn {
		t.Fatalf("recover persisted turn: %q", recovered)
	}
	for _, state := range []string{"released", "already-released"} {
		response := f.operator(t, false, 0, "operator-release", newFile, id, recovered)["OperatorLock"]
		want = map[string]any{"ID": id, "File": newFile, "Holder": "", "Turn": recovered, "LockPath": newFile + ".lock", "LockText": "", "State": state}
		if !reflect.DeepEqual(response, want) {
			t.Fatalf("release moved proof: %v", response)
		}
	}
	if _, err := os.Lstat(newFile + ".lock"); !os.IsNotExist(err) {
		t.Fatalf("new lock remains: %v", err)
	}
	// Old-path lock cleanup belongs to the editor's existing move journal.
	if err := os.Remove(lockPath); err != nil {
		t.Fatal(err)
	}
	newTurn := strings.Repeat("d", 32)
	newLock := f.operator(t, false, 0, "operator-acquire", newFile, id, newTurn)["OperatorLock"].(map[string]any)
	if response := f.operator(t, false, 3, "operator-release", newFile, id, recovered); response["Code"] != "operator-lock" {
		t.Fatalf("stale turn: %v", response)
	}
	if raw, err := os.ReadFile(newFile + ".lock"); err != nil || string(raw) != newLock["LockText"] {
		t.Fatalf("stale release changed new turn: %q %v", raw, err)
	}
	f.operator(t, false, 0, "operator-release", newFile, id, newTurn)
	assertStopped()
}
