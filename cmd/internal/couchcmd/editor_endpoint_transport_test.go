package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/editorendpoint"
)

type endpointTransportRuntime struct {
	Runtime
	ns couchcore.CouchNamespace
}

func (r endpointTransportRuntime) ResolveNamespace() (couchcore.CouchNamespace, error) {
	return r.ns, nil
}
func endpointTransportSetup(t *testing.T) (endpointTransportRuntime, editorendpoint.RegisterRequest) {
	t.Helper()
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	ns, err := couchcore.ExistingCouchNamespace(root)
	if err != nil {
		t.Fatal(err)
	}
	return endpointTransportRuntime{ns: ns}, editorendpoint.RegisterRequest{Version: 1, Namespace: root, DocumentID: strings.Repeat("a", 32), EditorIncarnation: strings.Repeat("b", 32), AttachID: "1", PID: 123, Socket: filepath.Join(root, "editor.sock"), Buffer: 7, Mode: "desktop"}
}
func endpointTransportRun(t *testing.T, inv cliInvocation, rt Runtime, input string, call func(context.Context, string, any, any) error) (int, editorendpoint.EndpointReply, string) {
	t.Helper()
	var out, errout bytes.Buffer
	code := runEditorEndpointCLI(inv, rt, strings.NewReader(input), &out, &errout, call)
	var reply editorendpoint.EndpointReply
	if err := json.Unmarshal(out.Bytes(), &reply); err != nil {
		t.Fatalf("non-JSON output %q: %v", out.String(), err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &object); err != nil {
		t.Fatal(err)
	}
	if _, ok := object["EditorEndpoint"]; !ok {
		t.Fatalf("missing explicit endpoint: %s", out.String())
	}
	return code, reply, errout.String()
}
func TestEditorEndpointTransportRejectsInputBeforeHandoff(t *testing.T) {
	rt, r := endpointTransportSetup(t)
	valid, _ := json.Marshal(r)
	for _, input := range []string{
		strings.Replace(string(valid), `"Version":1`, `"Version":1,"Version":1`, 1),
		strings.Replace(string(valid), `"Version":1`, `"version":1`, 1),
		strings.Replace(string(valid), `"PID":123`, `"PID":9999999999999999999999999999999`, 1),
		strings.Replace(string(valid), `"Version":1,`, "", 1),
		string(valid[:len(valid)-1]) + `,"Surprise":true}`,
		string(valid) + strings.Repeat(" ", 8193),
		strings.Replace(string(valid), `"Namespace":"`+r.Namespace+`"`, `"Namespace":"/wrong"`, 1),
	} {
		t.Run(fmt.Sprintf("input-%d", len(input)), func(t *testing.T) {
			calls := 0
			code, p, _ := endpointTransportRun(t, cliInvocation{args: []string{"register"}}, rt, input, func(context.Context, string, any, any) error { calls++; return nil })
			if code != 2 || p.Code != "invalid" || calls != 0 {
				t.Fatalf("code=%d reply=%+v calls=%d", code, p, calls)
			}
		})
	}
}
func TestEditorEndpointTransportOneExchangeAndReplyCodes(t *testing.T) {
	rt, r := endpointTransportSetup(t)
	input, _ := json.Marshal(r)
	for _, tt := range []struct {
		code string
		exit int
	}{{"ok", 0}, {"invalid", 2}, {"unavailable", 3}, {"capacity", 3}, {"superseded", 4}, {"stale", 4}, {"conflict", 4}, {"not-registered", 4}, {"desktop-active", 4}, {"uncertain", 5}} {
		t.Run(tt.code, func(t *testing.T) {
			calls := 0
			code, p, stderr := endpointTransportRun(t, cliInvocation{args: []string{"register"}}, rt, string(input), func(ctx context.Context, socket string, request, response any) error {
				calls++
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > 3*time.Second {
					t.Error("missing bounded deadline")
				}
				want, _ := couchmessage.SocketPath(r.Namespace, "editor-endpoint")
				if socket != want {
					t.Errorf("socket=%s want=%s", socket, want)
				}
				raw, _ := json.Marshal(editorendpoint.EndpointReply{Version: 1, Code: tt.code})
				return json.Unmarshal(raw, response)
			})
			if code != tt.exit || p.Code != tt.code || calls != 1 || stderr != "" {
				t.Fatalf("code=%d reply=%+v calls=%d stderr=%s", code, p, calls, stderr)
			}
		})
	}
}
func TestEditorEndpointTransportMalformedReplyAndErrors(t *testing.T) {
	rt, r := endpointTransportSetup(t)
	input, _ := json.Marshal(r)
	for _, raw := range []string{`{"Version":2,"Code":"ok","Error":"","EditorEndpoint":null}`, `{"Version":1,"Code":"surprise","Error":"","EditorEndpoint":null}`, `{"Version":1,"Code":"ok","Error":3,"EditorEndpoint":null}`, `{"Version":1,"Code":"ok","Error":""}`, `{"Version":1,"Code":"ok","Error":"","EditorEndpoint":null,"Padding":"` + strings.Repeat("x", 1<<20) + `"}`} {
		code, p, _ := endpointTransportRun(t, cliInvocation{args: []string{"register"}}, rt, string(input), func(_ context.Context, _ string, _, response any) error { return json.Unmarshal([]byte(raw), response) })
		if code != 5 || p.Code != "uncertain" {
			t.Fatalf("malformed reply accepted: %d %+v", code, p)
		}
	}
	for _, tt := range []struct {
		err  error
		exit int
		code string
	}{
		{&net.OpError{Op: "dial", Net: "unix", Err: syscall.ENOENT}, 3, "unavailable"},
		{&net.OpError{Op: "read", Net: "unix", Err: syscall.ECONNRESET}, 5, "uncertain"},
		{io.EOF, 5, "uncertain"}, {context.DeadlineExceeded, 5, "uncertain"}, {errors.New("private details"), 5, "uncertain"},
	} {
		calls := 0
		code, p, stderr := endpointTransportRun(t, cliInvocation{args: []string{"register"}}, rt, string(input), func(context.Context, string, any, any) error { calls++; return tt.err })
		if code != tt.exit || p.Code != tt.code || calls != 1 || strings.Contains(stderr, "private details") {
			t.Fatalf("error %v: %d %+v calls=%d stderr=%s", tt.err, code, p, calls, stderr)
		}
	}
}
func TestEditorEndpointTransportPrivateUnixAndBatch(t *testing.T) {
	rt, r := endpointTransportSetup(t)
	// No spawned process or operator runtime is involved: the private listener
	// and canonical namespace are both scoped to this test.
	socket, err := couchmessage.SocketPath(r.Namespace, "editor-endpoint")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var requests []endpointIPCRequest
	server, err := couchmessage.StartServer(t.Context(), socket, func(_ context.Context, raw []byte) ([]byte, error) {
		var req endpointIPCRequest
		if e := json.Unmarshal(raw, &req); e != nil {
			return nil, e
		}
		mu.Lock()
		requests = append(requests, req)
		mu.Unlock()
		return json.Marshal(editorendpoint.EndpointReply{Version: 1, Code: "ok"})
	})
	if err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(r)
	code, _, _ := endpointTransportRun(t, cliInvocation{args: []string{"register"}}, rt, string(input), couchmessage.Call)
	if code != 0 {
		t.Fatalf("register exit %d", code)
	}
	code, _, _ = endpointTransportRun(t, cliInvocation{args: []string{"lookup", r.Namespace, r.DocumentID}}, rt, "", couchmessage.Call)
	if code != 0 {
		t.Fatalf("lookup exit %d", code)
	}
	u := editorendpoint.UnregisterRequest{Version: 1, Namespace: r.Namespace, EditorIncarnation: r.EditorIncarnation, PID: r.PID}
	for i := range 256 {
		u.Entries = append(u.Entries, editorendpoint.UnregisterEntry{DocumentID: fmt.Sprintf("%032x", i), AttachID: "1", BindingGeneration: strings.Repeat("c", 32)})
	}
	input, _ = json.Marshal(u)
	code, _, _ = endpointTransportRun(t, cliInvocation{args: []string{"unregister"}}, rt, string(input), couchmessage.Call)
	if code != 0 {
		t.Fatalf("unregister exit %d", code)
	}
	mu.Lock()
	got := append([]endpointIPCRequest(nil), requests...)
	mu.Unlock()
	if len(got) != 3 || got[0].Operation != "register" || got[0].Register.DocumentID != r.DocumentID || got[1].Namespace != r.Namespace || got[1].DocumentID != r.DocumentID || len(got[2].Unregister.Entries) != 256 {
		t.Fatalf("requests %+v", got)
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadDir(r.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	code, p, _ := endpointTransportRun(t, cliInvocation{args: []string{"lookup", r.Namespace, r.DocumentID}}, rt, "", couchmessage.Call)
	after, err := os.ReadDir(r.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	if code != 3 || p.Code != "unavailable" || len(before) != len(after) {
		t.Fatalf("offline code=%d reply=%+v files=%d/%d", code, p, len(before), len(after))
	}
}

func TestEditorEndpointTransportUnregisterRejectsNestedAndOversizedInput(t *testing.T) {
	rt, r := endpointTransportSetup(t)
	u := editorendpoint.UnregisterRequest{Version: 1, Namespace: r.Namespace, EditorIncarnation: r.EditorIncarnation, PID: r.PID, Entries: []editorendpoint.UnregisterEntry{{DocumentID: r.DocumentID, AttachID: "1", BindingGeneration: strings.Repeat("c", 32)}}}
	valid, _ := json.Marshal(u)
	for _, input := range []string{
		strings.Replace(string(valid), `"DocumentID":`, `"documentid":`, 1),
		strings.Replace(string(valid), `"AttachID":"1"`, `"AttachID":"1","AttachID":"2"`, 1),
		strings.Replace(string(valid), `"AttachID":"1"`, `"AttachID":"1","Unknown":true`, 1),
		string(valid) + strings.Repeat(" ", (256<<10)+1),
	} {
		calls := 0
		code, p, _ := endpointTransportRun(t, cliInvocation{args: []string{"unregister"}}, rt, input, func(context.Context, string, any, any) error { calls++; return nil })
		if code != 2 || p.Code != "invalid" || calls != 0 {
			t.Fatalf("code=%d reply=%+v calls=%d", code, p, calls)
		}
	}
	u.Entries = nil
	for i := range 257 {
		u.Entries = append(u.Entries, editorendpoint.UnregisterEntry{DocumentID: fmt.Sprintf("%032x", i), AttachID: "1", BindingGeneration: strings.Repeat("c", 32)})
	}
	input, _ := json.Marshal(u)
	calls := 0
	code, p, _ := endpointTransportRun(t, cliInvocation{args: []string{"unregister"}}, rt, string(input), func(context.Context, string, any, any) error { calls++; return nil })
	if code != 2 || p.Code != "invalid" || calls != 0 {
		t.Fatalf("257 entries: code=%d reply=%+v calls=%d", code, p, calls)
	}
}

func TestEditorEndpointTransportAcceptsAdditiveReplyFields(t *testing.T) {
	rt, r := endpointTransportSetup(t)
	code, p, _ := endpointTransportRun(t, cliInvocation{args: []string{"lookup", r.Namespace, r.DocumentID}}, rt, "", func(_ context.Context, _ string, _, response any) error {
		return json.Unmarshal([]byte(`{"Version":1,"Code":"ok","Error":"","EditorEndpoint":null,"Future":{"Version":99}}`), response)
	})
	if code != 0 || p.Code != "ok" {
		t.Fatalf("additive reply refused: %d %+v", code, p)
	}
}

func TestEditorEndpointTransportBoundsReencodedReply(t *testing.T) {
	rt, r := endpointTransportSetup(t)
	code, p, _ := endpointTransportRun(t, cliInvocation{args: []string{"lookup", r.Namespace, r.DocumentID}}, rt, "", func(_ context.Context, _ string, _, response any) error {
		// JSON's default HTML escaping expands this bounded wire response sixfold.
		raw := `{"Version":1,"Code":"unavailable","Error":"` + strings.Repeat("<", 200000) + `","EditorEndpoint":null}`
		return json.Unmarshal([]byte(raw), response)
	})
	if code != 5 || p.Code != "uncertain" {
		t.Fatalf("unbounded printed reply accepted: exit=%d Code=%q ErrorBytes=%d", code, p.Code, len(p.Error))
	}
}

func TestEditorEndpointTransportRejectsMismatchedBindingReply(t *testing.T) {
	rt, r := endpointTransportSetup(t)
	for _, differentNamespace := range []bool{false, true} {
		b := editorendpoint.EndpointBinding{Namespace: r.Namespace, DocumentID: r.DocumentID, EditorIncarnation: r.EditorIncarnation, AttachID: r.AttachID, PID: r.PID, ProcessIdentity: "birth", Socket: r.Socket, Buffer: r.Buffer, Mode: r.Mode, BindingGeneration: strings.Repeat("c", 32), File: filepath.Join(r.Namespace, "document.md")}
		if differentNamespace {
			b.Namespace = "/different"
		} else {
			b.DocumentID = strings.Repeat("d", 32)
		}
		code, p, _ := endpointTransportRun(t, cliInvocation{args: []string{"lookup", r.Namespace, r.DocumentID}}, rt, "", func(_ context.Context, _ string, _, response any) error {
			raw, _ := json.Marshal(editorendpoint.EndpointReply{Version: 1, Code: "ok", EditorEndpoint: &b})
			return json.Unmarshal(raw, response)
		})
		if code != 5 || p.Code != "uncertain" {
			t.Fatalf("mismatched binding accepted: %d %s", code, p.Code)
		}
	}
}
