package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/durablefile"
	"github.com/xianxu/pair/cmd/internal/editorendpoint"
)

func TestEditorEndpointServiceRefusesNamespaceAndMalformedBeforeAuthority(t *testing.T) {
	s := &editorEndpointService{namespace: "/physical/store"}
	for _, raw := range []string{
		`{"Operation":"lookup","Namespace":"/other","DocumentID":"0123456789abcdef0123456789abcdef"}`,
		`{"Operation":"lookup","Operation":"register"}`,
		`{"Operation":"execute","Namespace":"/physical/store"}`,
		`{"operation":"lookup","Namespace":"/physical/store","DocumentID":"0123456789abcdef0123456789abcdef"}`,
	} {
		out, err := s.handle(context.Background(), []byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		var reply struct {
			Code           string
			Version        int
			EditorEndpoint any
		}
		if err = json.Unmarshal(out, &reply); err != nil {
			t.Fatal(err)
		}
		if reply.Code != "invalid" || reply.Version != 1 || reply.EditorEndpoint != nil {
			t.Fatalf("accepted malformed/foreign request: %s", out)
		}
	}
}

// Use the real producer, its private Unix transport and the CLI reader together:
// a server-only outcome code must never turn confirmed absence into uncertainty.
func TestEditorEndpointServiceEmptyLookupThroughCLI(t *testing.T) {
	rt, request := endpointTransportSetup(t)
	if err := os.Chmod(request.Namespace, 0700); err != nil {
		t.Fatal(err)
	}
	service, err := startEditorEndpointService(context.Background(), request.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(service.Close)
	code, reply, stderr := endpointTransportRun(t, cliInvocation{args: []string{"lookup", request.Namespace, request.DocumentID}}, rt, "", couchmessage.Call)
	if code != 0 || reply.Code != "ok" || reply.EditorEndpoint != nil || reply.Error != "" || stderr != "" {
		t.Fatalf("confirmed empty lookup: exit=%d reply=%+v stderr=%q", code, reply, stderr)
	}
}

func TestEditorEndpointServiceDamagedStateRefusesThroughCLI(t *testing.T) {
	rt, request := endpointTransportSetup(t)
	if err := os.Chmod(request.Namespace, 0700); err != nil {
		t.Fatal(err)
	}
	paths, err := artifactpath.EditorEndpoints(request.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(paths.State(), []byte("damaged authority"), 0600); err != nil {
		t.Fatal(err)
	}
	service, err := startEditorEndpointService(context.Background(), request.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(service.Close)
	code, reply, _ := endpointTransportRun(t, cliInvocation{args: []string{"lookup", request.Namespace, request.DocumentID}}, rt, "", couchmessage.Call)
	if code != 3 || reply.Code != "unavailable" || reply.EditorEndpoint != nil {
		t.Fatalf("damaged authority: exit=%d reply=%+v", code, reply)
	}
	raw, err := os.ReadFile(paths.State())
	if err != nil || string(raw) != "damaged authority" {
		t.Fatalf("authority changed: %q %v", raw, err)
	}
}

// endpointServiceEditor supplies stateful observations while the actual owner,
// durable store, service decoder, framed Unix transport and CLI all run intact.
// The editor's private listener supplies real socket ownership metadata; this
// test deliberately leaves native Neovim/process behavior to its conformance tests.
type endpointServiceEditor struct {
	mu                                sync.Mutex
	documents                         map[string]editorendpoint.EditorDescription
	birth                             string
	reads, describes, writes, entropy int
	fail, failAfter                   bool
}

func (e *endpointServiceEditor) ReadIdentity(int) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.reads++
	return e.birth, nil
}
func (e *endpointServiceEditor) Describe(ctx context.Context, r editorendpoint.RegisterRequest) (*editorendpoint.EditorDescription, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.describes++
	d, ok := e.documents[r.DocumentID]
	if !ok {
		return nil, nil
	}
	return &d, nil
}
func (*endpointServiceEditor) Superseded(context.Context, editorendpoint.EndpointBinding) error {
	return nil
}
func (e *endpointServiceEditor) write(path string, raw []byte, stage string) error {
	e.mu.Lock()
	e.writes++
	fail, after := e.fail, e.failAfter
	e.mu.Unlock()
	if fail {
		return errors.New("injected publication failure")
	}
	if err := durablefile.WriteAtomicStaged(path, raw, stage); err != nil {
		return err
	}
	if after {
		return errors.New("injected lost publication acknowledgement")
	}
	return nil
}
func (e *endpointServiceEditor) generation() (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.entropy++
	return fmt.Sprintf("%032x", e.entropy), nil
}

func TestEditorEndpointServiceOwnerLifecycleThroughCLI(t *testing.T) {
	for _, after := range []bool{false, true} {
		t.Run(fmt.Sprintf("publication-fails-after-write=%v", after), func(t *testing.T) {
			rt, request := endpointTransportSetup(t)
			if err := os.Chmod(request.Namespace, 0700); err != nil {
				t.Fatal(err)
			}
			// A short physical temp directory keeps the editor-owned Unix address
			// within the native socket limit even when the namespace test name is long.
			runtimeDir, err := os.MkdirTemp("", "eecs-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := os.RemoveAll(runtimeDir); err != nil {
					t.Error(err)
				}
			})
			physical, err := filepath.EvalSymlinks(runtimeDir)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.Chmod(physical, 0700); err != nil {
				t.Fatal(err)
			}
			request.Socket = filepath.Join(physical, "editor.sock")
			listener, err := net.Listen("unix", request.Socket)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = listener.Close() })
			if err = editorendpoint.ValidateEndpointSocket(request.Socket); err != nil {
				t.Fatal(err)
			}
			paths, err := artifactpath.EditorEndpoints(request.Namespace)
			if err != nil {
				t.Fatal(err)
			}
			editor := &endpointServiceEditor{birth: "test-native-birth", documents: map[string]editorendpoint.EditorDescription{}}
			requests := []editorendpoint.RegisterRequest{request, request}
			requests[1].DocumentID = strings.Repeat("c", 32)
			requests[1].Buffer++
			for i, r := range requests {
				editor.documents[r.DocumentID] = editorendpoint.EditorDescription{Namespace: r.Namespace, DocumentID: r.DocumentID, EditorIncarnation: r.EditorIncarnation, AttachID: r.AttachID, PID: r.PID, Socket: r.Socket, Buffer: r.Buffer, Mode: r.Mode, File: filepath.Join(request.Namespace, fmt.Sprintf("document-%d.md", i))}
			}
			store := editorendpoint.EndpointStore{Namespace: request.Namespace, Path: paths.State(), Stage: paths.Stage(), WriteFile: editor.write}
			owner, err := editorendpoint.NewEndpointOwner(t.Context(), editorendpoint.EndpointConfig{Store: store, RPC: editor, Processes: editor, Resolve: func(_ context.Context, id string) (string, error) {
				editor.mu.Lock()
				defer editor.mu.Unlock()
				if d, ok := editor.documents[id]; ok {
					return d.File, nil
				}
				return "", editorendpoint.ErrDocumentNotRegistered
			}, Random: editor.generation})
			if err != nil {
				t.Fatal(err)
			}
			service := &editorEndpointService{namespace: request.Namespace, owner: owner}
			socket, err := couchmessage.SocketPath(request.Namespace, "editor-endpoint")
			if err != nil {
				t.Fatal(err)
			}
			service.server, err = couchmessage.StartServer(t.Context(), socket, service.handle)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(service.Close)
			invoke := func(operation string, payload any) editorendpoint.EndpointReply {
				t.Helper()
				var input []byte
				if payload != nil {
					var err error
					input, err = json.Marshal(payload)
					if err != nil {
						t.Fatal(err)
					}
				}
				args := []string{operation}
				if operation == "lookup" {
					args = append(args, request.Namespace, request.DocumentID)
				}
				code, reply, stderr := endpointTransportRun(t, cliInvocation{args: args}, rt, string(input), couchmessage.Call)
				if code != 0 || reply.Code != "ok" || reply.Version != 1 || reply.Error != "" || stderr != "" {
					t.Fatalf("%s: exit=%d reply=%+v stderr=%q", operation, code, reply, stderr)
				}
				return reply
			}
			first := invoke("register", requests[0])
			if !first.Changed || first.EditorEndpoint == nil || first.EditorEndpoint.ProcessIdentity != editor.birth || first.EditorEndpoint.File != editor.documents[request.DocumentID].File {
				t.Fatalf("initial publication: %+v", first)
			}
			generation := first.EditorEndpoint.BindingGeneration
			retry := invoke("register", requests[0])
			if retry.Changed || retry.EditorEndpoint == nil || retry.EditorEndpoint.BindingGeneration != generation {
				t.Fatalf("exact retry: %+v", retry)
			}
			current := invoke("lookup", nil)
			if current.Changed || current.EditorEndpoint == nil || !reflect.DeepEqual(current.EditorEndpoint, first.EditorEndpoint) {
				t.Fatalf("lookup changed binding: %+v", current)
			}
			second := invoke("register", requests[1])
			if !second.Changed || second.EditorEndpoint == nil || second.EditorEndpoint.BindingGeneration == generation {
				t.Fatalf("second document publication: %+v", second)
			}
			state, err := store.Load()
			if err != nil || state.Revision != 2 || len(state.Records) != 2 {
				t.Fatalf("published state: %+v %v", state, err)
			}
			unregister := editorendpoint.UnregisterRequest{Version: 1, Namespace: request.Namespace, EditorIncarnation: request.EditorIncarnation, PID: request.PID, ProcessIdentity: editor.birth}
			for _, binding := range []*editorendpoint.EndpointBinding{first.EditorEndpoint, second.EditorEndpoint} {
				unregister.Entries = append(unregister.Entries, editorendpoint.UnregisterEntry{DocumentID: binding.DocumentID, AttachID: binding.AttachID, BindingGeneration: binding.BindingGeneration})
			}
			removed := invoke("unregister", unregister)
			if !removed.Changed || removed.EditorEndpoint != nil || len(removed.Results) != 2 {
				t.Fatalf("batch unregister: %+v", removed)
			}
			for _, result := range removed.Results {
				if result.Status != "removed" {
					t.Fatalf("batch result: %+v", result)
				}
			}
			empty := invoke("lookup", nil)
			if empty.Changed || empty.EditorEndpoint != nil {
				t.Fatalf("post-unregister lookup: %+v", empty)
			}
			state, err = store.Load()
			if err != nil || state.Revision != 3 {
				t.Fatalf("retirement state: %+v %v", state, err)
			}
			for _, record := range state.Records {
				if editorendpoint.CurrentBinding(record) != nil {
					t.Fatal("unregister left durable current binding")
				}
			}
			editor.mu.Lock()
			if editor.writes != 3 || editor.entropy != 2 || editor.reads != 8 || editor.describes != 4 {
				t.Errorf("unexpected work writes=%d entropy=%d reads=%d describe=%d", editor.writes, editor.entropy, editor.reads, editor.describes)
			}
			editor.fail = !after
			editor.failAfter = after
			replacement := requests[0]
			replacement.AttachID = "2"
			d := editor.documents[replacement.DocumentID]
			d.AttachID = "2"
			editor.documents[replacement.DocumentID] = d
			editor.mu.Unlock()
			input, err := json.Marshal(replacement)
			if err != nil {
				t.Fatal(err)
			}
			code, reply, stderr := endpointTransportRun(t, cliInvocation{args: []string{"register"}}, rt, string(input), couchmessage.Call)
			if code != 5 || reply.Code != "uncertain" || reply.Changed || reply.EditorEndpoint != nil || stderr != "" {
				t.Fatalf("publication failure: exit=%d reply=%+v stderr=%q", code, reply, stderr)
			}
			code, reply, _ = endpointTransportRun(t, cliInvocation{args: []string{"lookup", request.Namespace, request.DocumentID}}, rt, "", couchmessage.Call)
			if code != 3 || reply.Code != "unavailable" || reply.EditorEndpoint != nil {
				t.Fatalf("poisoned authority: exit=%d reply=%+v", code, reply)
			}
			persisted, err := store.Load()
			if err != nil {
				t.Fatal(err)
			}
			wantRevision := uint64(3)
			if after {
				wantRevision++
			}
			if persisted.Revision != wantRevision {
				t.Fatalf("failed publication evidence revision=%d want=%d", persisted.Revision, wantRevision)
			}
		})
	}
}
