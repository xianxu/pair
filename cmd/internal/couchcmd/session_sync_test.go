package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestParseSessionSyncForms(t *testing.T) {
	for _, c := range []struct {
		args []string
		want sessionSyncArgs
	}{
		{[]string{"diff", "f.md"}, sessionSyncArgs{verb: "diff", file: "f.md"}},
		{[]string{"status", "--json", "f.md"}, sessionSyncArgs{verb: "status", file: "f.md", json: true}},
		{[]string{"write", "--section", "--odd heading", "b", "f.md", "--fold", "a, b", "--if-hash", "h1", "--accept-gap"},
			sessionSyncArgs{verb: "write", section: "--odd heading", bodyFile: "b", file: "f.md", ifHash: "h1", fold: []string{"a", "b"}, accepts: sessionsync.Accepts{Gap: true}}},
		{[]string{"write", "--all", "b", "f.md", "--if-hash", "h"}, sessionSyncArgs{verb: "write", all: true, bodyFile: "b", file: "f.md", ifHash: "h"}},
		{[]string{"read", "--section", "Needs you", "f.md", "--json"}, sessionSyncArgs{verb: "read", section: "Needs you", file: "f.md", json: true}},
		{[]string{"read", "--all", "f.md"}, sessionSyncArgs{verb: "read", all: true, file: "f.md"}},
		{[]string{"apply", "e.json", "f.md", "--if-hash", "h", "--fold", "a", "--json"}, sessionSyncArgs{verb: "apply", bodyFile: "e.json", file: "f.md", ifHash: "h", fold: []string{"a"}, json: true}},
		{[]string{"round", "f.md", "--json"}, sessionSyncArgs{verb: "round", file: "f.md", json: true}},
		// release is a retired no-op that still takes its old flags.
		{[]string{"release", "f.md", "--fold", "x"}, sessionSyncArgs{verb: "release", file: "f.md", fold: []string{"x"}}},
		{[]string{"release", "--accept-edits", "--accept-gap", "f.md"}, sessionSyncArgs{verb: "release", file: "f.md", accepts: sessionsync.Accepts{Gap: true}}},
	} {
		inv, err := ParseCLI(append([]string{"--session-sync"}, c.args...), couchcore.Operations())
		if err != nil {
			t.Fatalf("%v: %v", c.args, err)
		}
		if inv.kind != cliSessionSync || !reflect.DeepEqual(*inv.sessionSync, c.want) {
			t.Fatalf("%v: got %+v", c.args, inv.sessionSync)
		}
	}
	for _, bad := range [][]string{
		{}, {"nope", "f"}, {"diff"}, {"diff", "a", "b"}, {"status", "--fold", "x", "f"},
		{"write", "b", "f", "--if-hash", "h"}, {"write", "--all", "--section", "h", "b", "f", "--if-hash", "h"},
		{"write", "--all", "f", "--if-hash", "h"}, {"write", "--section"}, {"release", "f", "--fold"},
		{"status", "--json", "--json", "f"},
		// An unconditional write, and the retired --accept-edits.
		{"write", "--all", "b", "f"}, {"write", "--all", "b", "f", "--if-hash", "h", "--accept-edits"},
		{"read", "f"}, {"read", "--all", "--section", "h", "f"}, {"read", "--all", "f", "--if-hash", "h"},
		// apply needs the whole-file hash and both paths; round answers only as JSON.
		{"apply", "e.json", "f"}, {"apply", "f", "--if-hash", "h"}, {"apply", "--all", "e.json", "f", "--if-hash", "h"},
		{"round", "f"}, {"round", "--if-hash", "h", "f", "--json"},
	} {
		if _, err := ParseCLI(append([]string{"--session-sync"}, bad...), couchcore.Operations()); err == nil {
			t.Fatalf("%v must be a usage error", bad)
		}
	}
}

// sessionSyncRT is a runtime whose namespace is a recording Couch store,
// plus a session-sync file owned by ops:0.
func sessionSyncRT(t *testing.T) (testRT, string) {
	rt := newRT(t)
	store := t.TempDir()
	ns, err := couchcore.ResolveCouchNamespace(store, "/")
	if err != nil {
		t.Fatal(err)
	}
	rt.namespace = ns
	ledger := &sessionsync.Ledger{Dir: sessionsync.LedgerDir(ns.Dir())}
	if err := ledger.MarkRecording(sessionsync.Recording{PID: 1, Started: time.Now()}); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "tl.md")
	doc := "---\ntype: session-sync\nowner: ops:0\n---\n\n## Needs you\n\nold\n"
	os.WriteFile(file, []byte(doc), 0o644)
	os.WriteFile(sessionsync.PathsOf(file).Shadow, []byte(doc), 0o644)
	// After the shadow: its mtime is the owed horizon of a file never
	// written through this verb.
	if err := ledger.Append("ops:0", sessionsync.Entry{ID: "d1", From: "pair:5", At: time.Now().Add(time.Second), Head: "done"}); err != nil {
		t.Fatal(err)
	}
	return rt, file
}

func runSS(t *testing.T, rt testRT, args ...string) (int, string, string) {
	var out, errout bytes.Buffer
	code := RunWithRuntime(append([]string{"--session-sync"}, args...), strings.NewReader(""), &out, &errout, rt)
	return code, out.String(), errout.String()
}

func TestSessionSyncRegistryCLIContract(t *testing.T) {
	rt, _ := sessionSyncRT(t)
	file := filepath.Join(t.TempDir(), "ordinary note.md")
	if err := os.WriteFile(file, []byte("## Work\n\nplain\n"), 0600); err != nil {
		t.Fatal(err)
	}
	call := func(args ...string) map[string]any {
		t.Helper()
		code, out, errout := runSS(t, rt, args...)
		if code != 0 {
			t.Fatalf("%v: exit%d %s %s", args, code, out, errout)
		}
		var response map[string]any
		if err := json.Unmarshal([]byte(out), &response); err != nil {
			t.Fatal(err)
		}
		if response["Code"] != "ok" {
			t.Fatalf("response %+v", response)
		}
		return response
	}
	if response := call("lookup", file, "--json"); response["Registration"] != nil {
		t.Fatalf("absence %+v", response)
	}
	first := call("register", file, "--owner", "ops:0", "--json")
	registration := first["Registration"].(map[string]any)
	if registration["Owner"] != "ops:0" || registration["RegisteredOwner"] != "ops:0" || registration["ID"] == "" || first["Changed"] != true {
		t.Fatalf("registration %+v", first)
	}
	if replay := call("register", file, "--owner", "ops:0", "--json"); replay["Changed"] != false || replay["Registration"].(map[string]any)["ID"] != registration["ID"] {
		t.Fatalf("replay %+v", replay)
	}
	if response := call("list", "--json"); len(response["Registrations"].([]any)) != 1 {
		t.Fatalf("list %+v", response)
	}
	status := call("status", file, "--json")
	if status["Status"].(map[string]any)["Owner"] != "ops:0" {
		t.Fatalf("status %+v", status)
	}
	newFile := filepath.Join(filepath.Dir(file), "renamed.md")
	if err := os.Rename(file, newFile); err != nil {
		t.Fatal(err)
	}
	move := call("rename", file, newFile, "--json")
	if move["Registration"].(map[string]any)["ID"] != registration["ID"] {
		t.Fatalf("move %+v", move)
	}
	call("forget", newFile, "--json")
	if response := call("lookup", newFile, "--json"); response["Registration"] != nil {
		t.Fatalf("forgotten %+v", response)
	}
}

func TestSessionSyncRegistryLookupFailureCannotBeAbsence(t *testing.T) {
	rt, _ := sessionSyncRT(t)
	file := filepath.Join(t.TempDir(), "note.md")
	if err := os.WriteFile(file, []byte("note"), 0600); err != nil {
		t.Fatal(err)
	}
	if code, out, errout := runSS(t, rt, "register", file, "--owner", "ops:0", "--json"); code != 0 {
		t.Fatalf("register %s %s", out, errout)
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runSS(t, rt, "lookup", file, "--json")
	if code == 0 {
		t.Fatalf("unavailable succeeded: %s", out)
	}
	var response map[string]any
	if err := json.Unmarshal([]byte(out), &response); err != nil {
		t.Fatal(err)
	}
	if response["Code"] == "ok" || response["Error"] == "" {
		t.Fatalf("unavailable: %+v", response)
	}
}

func TestSessionSyncLookupJSONUsageFailureIsUnavailable(t *testing.T) {
	rt, _ := sessionSyncRT(t)
	code, out, _ := runSS(t, rt, "lookup", "--json")
	if code == 0 {
		t.Fatal("invalid lookup succeeded")
	}
	var response map[string]any
	if err := json.Unmarshal([]byte(out), &response); err != nil {
		t.Fatalf("missing refusal envelope: %q: %v", out, err)
	}
	if response["Code"] == "ok" || response["Error"] == "" {
		t.Fatalf("invalid lookup looked absent: %+v", response)
	}
}

func TestSessionSyncStatusJSONIsTheAgreedEnvelope(t *testing.T) {
	rt, file := sessionSyncRT(t)
	code, out, errout := runSS(t, rt, "status", "--json", file)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errout)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if got["Code"] != "ok" || got["Error"] != "" {
		t.Fatalf("got %v", got)
	}
	st := got["Status"].(map[string]any)
	keys := []string{}
	for k := range st {
		keys = append(keys, k)
	}
	for _, k := range []string{"File", "Owner", "Holder", "LastWrite", "Owed"} {
		if _, ok := st[k]; !ok {
			t.Fatalf("Status lacks %s: %v", k, keys)
		}
	}
	if st["LastWrite"] != nil || st["Holder"] != "" || st["Owner"] != "ops:0" || st["File"] != file {
		t.Fatalf("status %v", st)
	}
	owed := st["Owed"].([]any)
	row := owed[0].(map[string]any)
	if len(owed) != 1 || row["ID"] != "d1" || row["Kind"] != "delivery" || row["From"] != "pair:5" {
		t.Fatalf("owed %v", owed)
	}
	if _, err := time.Parse(time.RFC3339, row["At"].(string)); err != nil {
		t.Fatalf("At is RFC3339: %v", err)
	}
}

func TestSessionSyncRefusalsExitNonZeroWithACode(t *testing.T) {
	rt, file := sessionSyncRT(t)
	os.WriteFile(file+".lock", []byte("holder: operator\n"), 0o644)
	body := filepath.Join(t.TempDir(), "body")
	os.WriteFile(body, []byte("new"), 0o644)
	code, _, errout := runSS(t, rt, "write", "--section", "Needs you", body, file, "--if-hash", sessionSyncSectionHash(t, rt, file, "Needs you"))
	if code != sessionsync.ExitOperatorLock || !strings.Contains(errout, "operator holds the lock") {
		t.Fatalf("exit %d: %s", code, errout)
	}
	os.Remove(file + ".lock")
	os.WriteFile(file, []byte("plain\n"), 0o644)
	code, out, _ := runSS(t, rt, "status", "--json", file)
	var got struct {
		Code, Error string
		Status      any
	}
	json.Unmarshal([]byte(out), &got)
	if code == 0 || got.Code != "refused" || got.Status != nil {
		t.Fatalf("exit %d out %s", code, out)
	}
}

func TestSessionSyncWriteFoldsAndStatusTextShowsIt(t *testing.T) {
	rt, file := sessionSyncRT(t)
	_, out, _ := runSS(t, rt, "status", file)
	if !strings.Contains(out, "owed: 1") || !strings.Contains(out, "d1 from pair:5") {
		t.Fatalf("status text %q", out)
	}
	body := filepath.Join(t.TempDir(), "body")
	os.WriteFile(body, []byte("handled d1"), 0o644)
	_, out, _ = runSS(t, rt, "read", "--section", "Needs you", file)
	hash, rest, _ := strings.Cut(strings.TrimPrefix(out, "hash: "), "\n")
	if rest != "\nold\n" || hash != sessionSyncSectionHash(t, rt, file, "Needs you") {
		t.Fatalf("read text %q", out)
	}
	code, out, errout := runSS(t, rt, "write", "--section", "Needs you", body, file, "--fold", "d1", "--if-hash", hash)
	if code != 0 || !strings.Contains(out, "wrote ## Needs you") || !strings.Contains(out, "folded d1") ||
		!strings.Contains(out, "hash: "+sessionSyncSectionHash(t, rt, file, "Needs you")) {
		t.Fatalf("exit %d out %q err %q", code, out, errout)
	}
	if _, out, _ = runSS(t, rt, "status", file); !strings.Contains(out, "owed: 0") {
		t.Fatalf("status text %q", out)
	}
}

func TestSessionSyncStaleAndReadJSON(t *testing.T) {
	rt, file := sessionSyncRT(t)
	code, out, _ := runSS(t, rt, "read", "--section", "Needs you", "--json", file)
	var got struct {
		Code string
		Read sessionsync.ReadResult
		Hash string
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil || code != 0 || got.Code != "ok" ||
		got.Read.Body != "\nold\n" || len(got.Read.Hash) != 16 || got.Read.Editorial == nil || got.Read.EditorialCoordinates != "document-byte-offsets" || got.Read.Section != "Needs you" {
		t.Fatalf("exit %d out %s", code, out)
	}
	body := filepath.Join(t.TempDir(), "body")
	os.WriteFile(body, []byte("new"), 0o644)
	code, out, errout := runSS(t, rt, "write", "--section", "Needs you", body, file, "--if-hash", "0000000000000000")
	if code != sessionsync.ExitStale || out != "hash: "+got.Read.Hash+"\n" || !strings.Contains(errout, "changed since you read it") {
		t.Fatalf("exit %d out %q err %q", code, out, errout)
	}
}

func TestSessionSyncReleaseIsARetiredNoOp(t *testing.T) {
	rt, file := sessionSyncRT(t)
	before, _ := os.ReadFile(file)
	code, out, errout := runSS(t, rt, "release", file, "--accept-edits")
	after, _ := os.ReadFile(file)
	if code != 0 || out != "" || !strings.Contains(errout, "retired") || string(before) != string(after) {
		t.Fatalf("exit %d out %q err %q", code, out, errout)
	}
	// A requested fold that does not happen is a failure, not a silent 0.
	if code, _, errout = runSS(t, rt, "release", file, "--fold", "d1"); code == 0 || !strings.Contains(errout, "write --fold") {
		t.Fatalf("exit %d err %q", code, errout)
	}
	if _, out, _ = runSS(t, rt, "status", file); !strings.Contains(out, "owed: 1") || !strings.Contains(out, "lock: none") {
		t.Fatalf("release must fold nothing: %q", out)
	}
}

func TestSessionSyncWriteTelemetryBestEffort(t *testing.T) {
	for _, mode := range []string{"ok", "older", "missing", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			rt, file := sessionSyncRT(t)
			rt.env = messageRuntime().env
			rt.env["COUCH_STORE_DIR"] = rt.namespace.Dir()
			socket, err := couchmessage.SocketPath(rt.namespace.Dir(), "broker")
			if err != nil {
				t.Fatal(err)
			}
			var calls atomic.Int32
			requests := make(chan couchmessage.Request, 4)
			release := make(chan struct{})
			returned := make(chan struct{})
			if mode != "missing" {
				server, err := couchmessage.StartServer(context.Background(), socket, func(ctx context.Context, raw []byte) ([]byte, error) {
					calls.Add(1)
					var req couchmessage.Request
					if err := json.Unmarshal(raw, &req); err != nil {
						return nil, err
					}
					requests <- req
					defer close(returned)
					if mode == "timeout" {
						<-release
					}
					code := "ok"
					if mode == "older" {
						code = "invalid-request"
					}
					return json.Marshal(couchmessage.Response{Code: code})
				})
				if err != nil {
					t.Fatal(err)
				}
				defer server.Close()
				defer close(release)
			}
			body := filepath.Join(t.TempDir(), "body")
			os.WriteFile(body, []byte("handled d1"), 0600)
			type result struct {
				code        int
				out, errout string
			}
			done := make(chan result, 1)
			hash := sessionSyncSectionHash(t, rt, file, "Needs you")
			go func() {
				code, out, errout := runSS(t, rt, "write", "--section", "Needs you", body, file, "--fold", "d1", "--if-hash", hash)
				done <- result{code, out, errout}
			}()
			var got result
			select {
			case got = <-done:
			case <-testwait.After(t):
				t.Fatal("session write waited for the unreleased telemetry handler")
			}
			code, out, errout := got.code, got.out, got.errout
			if mode == "timeout" {
				select {
				case <-returned:
					t.Fatal("write did not finish while telemetry was blocked")
				default:
				}
			}
			if code != 0 || errout != "" || !strings.Contains(out, "wrote ## Needs you") {
				t.Fatalf("write %d %q %q", code, out, errout)
			}
			raw, _ := os.ReadFile(file)
			if !strings.Contains(string(raw), "handled d1") {
				t.Fatal("write lost")
			}
			if mode != "missing" {
				if calls.Load() != 1 {
					t.Fatalf("calls %d", calls.Load())
				}
				req := <-requests
				want := callerIdentity(rt, "session-write")
				want.Target = "ops:0"
				if !reflect.DeepEqual(req, want) {
					t.Fatalf("request %+v want %+v", req, want)
				}

				code, _, _ = runSS(t, rt, "write", "--section", "Needs you", body, file, "--if-hash", "stale")
				if code == 0 || calls.Load() != 1 {
					t.Fatalf("failed write reported: code=%d calls=%d", code, calls.Load())
				}
			}
		})
	}
}

func TestSessionWriteServiceAuthenticatesOwner(t *testing.T) {
	for _, fault := range []string{"none", "no-recorder", "owner", "nonce", "session", "replaced", "disconnected"} {
		t.Run(fault, func(t *testing.T) {
			r := &serviceRig{t: t, world: newMessageWorld(), slotGit: map[string]couchcore.SlotGitStatus{}}
			r.init()
			b := r.connect(0)
			sink := &factSink{}
			log := testFactLog(t, sink)
			if fault != "no-recorder" {
				r.s.facts.Store(log)
			}
			req := callerRequest(b, "session-write")
			req.Target = b.Slot
			switch fault {
			case "owner":
				req.Target = "pair:1"
			case "nonce":
				req.Nonce = "old"
			case "session":
				req.Session = "old"
			case "replaced":
				r.world.set(b, func(s *worldSlot) { s.pidFile++ })
			case "disconnected":
				r.disconnect(b)
			}
			var got couchmessage.Response
			if err := couchmessage.Call(context.Background(), r.brokerSock, req, &got); err != nil {
				t.Fatal(err)
			}
			if (got.Code == "ok") != (fault == "none" || fault == "no-recorder") {
				t.Fatalf("response %+v", got)
			}

			log.offer(supervisionFact{Event: "snapshot"})
			supervisionEventually(t, "sentinel not written", func() bool {
				for _, row := range sink.rows(t) {
					if row.Event == "snapshot" {
						return true
					}
				}
				return false
			})
			count := 0
			for _, row := range sink.rows(t) {
				if row.Event == "session_write" {
					count++
					if row.Subject != log.subject(b) || row.Incarnation != log.incarnation(b) {
						t.Fatalf("wrong identity: %+v", row)
					}
				}
			}
			want := 0
			if fault == "none" {
				want = 1
			}
			if count != want {
				t.Fatalf("writes=%d want=%d", count, want)
			}
		})
	}
}

// apply and round through the CLI: a good batch lands and leaves a round with
// its reason, a bad batch refuses with every bad record listed and nothing
// written, and round --json is exactly the parley.nvim#383 contract.
func TestSessionSyncApplyAndRoundCLIContract(t *testing.T) {
	rt, file := sessionSyncRT(t)
	edits := filepath.Join(t.TempDir(), "edits.json")
	os.WriteFile(edits, []byte(`{"records":[{"old":"old","occurrence":1,"new":"fresh","explain":"answer"}]}`), 0o644)
	raw, _ := os.ReadFile(file)
	code, out, errout := runSS(t, rt, "apply", edits, file, "--if-hash", sessionsync.Hash(string(raw)), "--json")
	if code != 0 {
		t.Fatalf("apply exit %d: %s %s", code, out, errout)
	}
	var applied map[string]any
	if err := json.Unmarshal([]byte(out), &applied); err != nil || applied["Code"] != "ok" || applied["Hash"] == "" {
		t.Fatalf("apply reply %q", out)
	}
	code, out, _ = runSS(t, rt, "round", file, "--json")
	var reply map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &reply); err != nil || code != 0 {
		t.Fatalf("round reply %q exit %d", out, code)
	}
	keys := []string{}
	for k := range reply {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if strings.Join(keys, ",") != "Cleared,Code,Error,Round" {
		t.Fatalf("round reply keys %v, want exactly Code, Error, Round, Cleared", keys)
	}
	var round sessionsync.Round
	if err := json.Unmarshal(reply["Round"], &round); err != nil || len(round.Spans) != 1 || round.Spans[0].Explain != "answer" {
		t.Fatalf("round %s", reply["Round"])
	}
	// A bad batch: exit 1, edits-invalid, each bad record listed, file intact.
	before, _ := os.ReadFile(file)
	os.WriteFile(edits, []byte(`[{"old":"fresh","new":"x"},{"old":"absent","new":"y"},{"old":"","new":"z"}]`), 0o644)
	code, out, _ = runSS(t, rt, "apply", edits, file, "--if-hash", sessionsync.Hash(string(before)), "--json")
	var refused struct {
		Code    string
		Invalid []sessionsync.BadEdit
	}
	if err := json.Unmarshal([]byte(out), &refused); err != nil || code != 1 || refused.Code != "edits-invalid" || len(refused.Invalid) != 2 {
		t.Fatalf("bad batch exit %d reply %q", code, out)
	}
	if after, _ := os.ReadFile(file); string(after) != string(before) {
		t.Fatal("a refused batch wrote")
	}
	// Stale: exit 6 with the current hash.
	code, out, _ = runSS(t, rt, "apply", edits, file, "--if-hash", "0000000000000000", "--json")
	if code != 6 || !strings.Contains(out, `"stale"`) {
		t.Fatalf("stale apply exit %d %q", code, out)
	}
	// A malformed sidecar: round-invalid in the same envelope.
	os.WriteFile(sessionsync.PathsOf(file).Round, []byte("{"), 0o644)
	code, out, _ = runSS(t, rt, "round", file, "--json")
	if code != 1 || !strings.Contains(out, `"round-invalid"`) || !strings.Contains(out, `"Round":null`) {
		t.Fatalf("malformed round exit %d %q", code, out)
	}
}

// section tokens bind policy as well as bytes; callers must reuse the opaque
// value the read endpoint returned rather than reconstruct a body-only hash.
func sessionSyncSectionHash(t *testing.T, rt testRT, file, heading string) string {
	t.Helper()
	code, out, errout := runSS(t, rt, "read", "--section", heading, file, "--json")
	var response struct {
		Code string
		Read sessionsync.ReadResult
	}
	if err := json.Unmarshal([]byte(out), &response); err != nil || code != 0 || response.Code != "ok" || len(response.Read.Hash) != 16 {
		t.Fatalf("section read %d %s %s", code, out, errout)
	}
	return response.Read.Hash
}
