package editorendpoint

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

type privateNativeEditor struct {
	root, socket, file string
	pid                int
	rpc                NvimRPC
	stop               func()
}

func newPrivateNativeEditor(t *testing.T, file string) *privateNativeEditor {
	t.Helper()
	program, err := exec.LookPath("nvim")
	if err != nil {
		t.Skip("installed nvim unavailable")
	}
	root := endpointTestRuntime(t)
	if file == "" {
		file = filepath.Join(root, "雪.txt")
		if err = os.WriteFile(file, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err = os.WriteFile(file+".second", []byte("second fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	f := &privateNativeEditor{root: root, socket: filepath.Join(root, "rpc"), file: file}
	script := filepath.Join(root, "fixture.lua")
	// OSProvisionIO assigns cmd.Env directly, rather than overlaying inherited
	// variables. These assertions qualify that boundary in the actual child.
	lua := `assert(vim.env.COUCH_TREE == nil, 'inherited Couch authority')
assert(vim.env.PAIR_SESSION_ID == nil, 'inherited Pair authority')
assert(vim.env.PAIR_SLUG == nil, 'inherited Pair identity')
vim.api.nvim_buf_set_name(1, vim.env.ENDPOINT_DOCUMENT)
local second = vim.api.nvim_create_buf(true, false)
assert(second == 2)
vim.api.nvim_buf_set_name(second, vim.env.ENDPOINT_DOCUMENT .. '.second')
package.preload['parley.remote_session'] = function()
 return {
 endpoint = function(q)
  if not vim.api.nvim_buf_is_valid(q.Buffer) then return nil end
  if q.AttachID == 'absent' or vim.fn.filereadable(vim.env.ENDPOINT_RETIRED) == 1 then return nil end
  if q.AttachID == 'error' then error('callback failed') end
  return { Namespace=q.Namespace, DocumentID=q.DocumentID,
   EditorIncarnation=q.EditorIncarnation, AttachID=q.AttachID,
   Buffer=q.Buffer, PID=vim.fn.getpid(), Socket=vim.v.servername,
   File=vim.api.nvim_buf_get_name(q.Buffer) }
 end,
 superseded = function(q)
  assert(type(q.BindingGeneration) == 'string' and #q.BindingGeneration > 0)
  local notice = assert(io.open(vim.env.ENDPOINT_NOTICE, 'w')); notice:write(q.BindingGeneration); notice:close()
  return nil
 end }
end
local ready = assert(io.open(vim.env.ENDPOINT_READY, 'w')); ready:write(vim.fn.getpid()); ready:close()
`
	if err = os.WriteFile(script, []byte(lua), 0600); err != nil {
		t.Fatal(err)
	}
	env := []string{"HOME=" + root, "XDG_CONFIG_HOME=" + root, "XDG_DATA_HOME=" + root, "XDG_STATE_HOME=" + root, "XDG_CACHE_HOME=" + root, "XDG_RUNTIME_DIR=" + root, "NVIM_LOG_FILE=" + filepath.Join(root, "log"), "PATH=/usr/bin:/bin", "TMPDIR=" + root, "ENDPOINT_FIXTURE=" + script, "ENDPOINT_READY=" + filepath.Join(root, "ready"), "ENDPOINT_DOCUMENT=" + file, "ENDPOINT_RETIRED=" + filepath.Join(root, "retired"), "ENDPOINT_NOTICE=" + filepath.Join(root, "notice")}
	native := couchcore.OSProvisionIO{Env: env}
	ctx, cancel := context.WithTimeout(context.Background(), testwait.Budget(t))
	done := make(chan error, 1)
	go func() {
		_, e := native.Run(ctx, couchcore.ProvisionCommand{Dir: root, Program: program, Args: []string{"--headless", "-u", "NONE", "-i", "NONE", "-n", "--listen", f.socket, "-c", "lua dofile(vim.env.ENDPOINT_FIXTURE)"}})
		done <- e
	}()
	var stopped sync.Once
	f.stop = func() {
		stopped.Do(func() {
			cancel()
			select {
			case <-done:
			case <-testwait.After(t):
				t.Error("owned Neovim did not join")
			}
		})
	}
	t.Cleanup(f.stop)
	testwait.Until(t, "private Neovim callback readiness", func() bool {
		select {
		case e := <-done:
			done <- e
			t.Fatalf("private fixture exited: %v", e)
		default:
		}
		raw, e := os.ReadFile(filepath.Join(root, "ready"))
		if e != nil {
			return false
		}
		pid, e := strconv.Atoi(strings.TrimSpace(string(raw)))
		if e != nil {
			return false
		}
		f.pid = pid
		return true
	})
	f.rpc = NvimRPC{Program: program, IO: native}
	return f
}
func poisonNativeParent(t *testing.T) {
	t.Helper()
	t.Setenv("COUCH_TREE", "foreign-couch")
	t.Setenv("PAIR_SESSION_ID", "foreign-session")
	t.Setenv("PAIR_SLUG", "foreign-pair")
}

func TestNvimPrivateNativeConformance(t *testing.T) {
	poisonNativeParent(t)
	f := newPrivateNativeEditor(t, "")
	ctx := context.Background()
	req := RegisterRequest{Namespace: "/private/quote' $()\n雪", DocumentID: "doc", EditorIncarnation: "inc", AttachID: "live", Buffer: 1, Socket: f.socket}
	got, err := f.rpc.Describe(ctx, req)
	if err != nil || got == nil || got.Namespace != req.Namespace || got.File != f.file || got.PID != f.pid {
		t.Fatalf("live response %+v %v", got, err)
	}
	req.AttachID = "absent"
	got, err = f.rpc.Describe(ctx, req)
	if err != nil || got != nil {
		t.Fatalf("Lua nil must become JSON null: %+v %v", got, err)
	}
	req.AttachID = "error"
	if _, err = f.rpc.Describe(ctx, req); err == nil {
		t.Fatal("callback error became positive absence")
	}
	if err = f.rpc.Superseded(ctx, EndpointBinding{Socket: f.socket, BindingGeneration: "generation"}); err != nil {
		t.Fatal(err)
	}
	if raw, e := os.ReadFile(filepath.Join(f.root, "notice")); e != nil || string(raw) != "generation" {
		t.Fatalf("native notice %q %v", raw, e)
	}
}

func TestEndpointOwnerTwoPrivateNativeEditors(t *testing.T) {
	poisonNativeParent(t)
	namespace := endpointTestRuntime(t)
	file := filepath.Join(namespace, "doc.md")
	if err := os.WriteFile(file, []byte("document"), 0600); err != nil {
		t.Fatal(err)
	}
	first, second := newPrivateNativeEditor(t, file), newPrivateNativeEditor(t, file)
	ctx := context.Background()
	generation := 0
	owner, err := NewEndpointOwner(ctx, EndpointConfig{Store: EndpointStore{Namespace: namespace, Path: filepath.Join(namespace, "state.json"), Stage: filepath.Join(namespace, "stage.json")}, RPC: first.rpc, Processes: procutil.Source{}, Resolve: func(context.Context, string) (string, error) { return file, nil }, Random: func() (string, error) { generation++; return fmt.Sprintf("%032x", generation), nil }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { owner.Close() })
	request := func(f *privateNativeEditor, inc, attach, mode string) RegisterRequest {
		return RegisterRequest{Version: 1, Namespace: namespace, DocumentID: docA, EditorIncarnation: inc, AttachID: attach, PID: f.pid, Socket: f.socket, Buffer: 1, Mode: mode}
	}
	a, b := request(first, incA, "1", "desktop"), request(second, incB, "1", "desktop")
	old := owner.Register(ctx, a)
	if old.Code != "ok" || old.EditorEndpoint == nil {
		t.Fatalf("first desktop: %+v", old)
	}
	current := owner.Register(ctx, b)
	if current.Code != "ok" || current.EditorEndpoint == nil || current.EditorEndpoint.PID != second.pid {
		t.Fatalf("second desktop: %+v", current)
	}
	if raw, e := os.ReadFile(filepath.Join(first.root, "notice")); e != nil || string(raw) != old.EditorEndpoint.BindingGeneration {
		t.Fatalf("displacement notice %q %v", raw, e)
	}
	stale := owner.Unregister(ctx, UnregisterRequest{Version: 1, Namespace: namespace, EditorIncarnation: incA, PID: first.pid, Entries: []UnregisterEntry{{DocumentID: docA, AttachID: a.AttachID, BindingGeneration: old.EditorEndpoint.BindingGeneration}}})
	if stale.Changed {
		t.Fatalf("stale unregister changed winner: %+v", stale)
	}
	headless := request(first, incA, "2", "headless")
	if p := owner.Register(ctx, headless); p.Code == "ok" {
		t.Fatalf("headless displaced desktop: %+v", p)
	}
	if p := owner.Lookup(ctx, namespace, docA); p.Code != "ok" || p.EditorEndpoint == nil || p.EditorEndpoint.PID != second.pid {
		t.Fatalf("desktop lookup: %+v", p)
	}
	if err = os.WriteFile(filepath.Join(second.root, "retired"), []byte("retired"), 0600); err != nil {
		t.Fatal(err)
	}
	if p := owner.Lookup(ctx, namespace, docA); p.Code != "ok" || p.EditorEndpoint != nil {
		t.Fatalf("live nil did not retire: %+v", p)
	}
	headless.AttachID = "3"
	if p := owner.Register(ctx, headless); p.Code != "ok" || p.EditorEndpoint == nil || p.EditorEndpoint.Mode != "headless" {
		t.Fatalf("headless fallback: %+v", p)
	}
	if err = os.Remove(filepath.Join(second.root, "retired")); err != nil {
		t.Fatal(err)
	}
	b.AttachID = "2"
	if p := owner.Register(ctx, b); p.Code != "ok" || p.EditorEndpoint == nil || p.EditorEndpoint.PID != second.pid {
		t.Fatalf("desktop displaced fallback: %+v", p)
	}
	saved, e := (EndpointStore{Namespace: namespace, Path: filepath.Join(namespace, "state.json"), Stage: filepath.Join(namespace, "stage.json")}).Load()
	if e != nil || selected(saved, docA) == nil || selected(saved, docA).PID != second.pid {
		t.Fatalf("durable native winner: %+v %v", saved, e)
	}
	beforeFirst := EndpointRecord{}
	for _, record := range saved.Records {
		if record.Binding.EditorIncarnation == incA {
			beforeFirst = record
		}
	}
	winner := *selected(saved, docA)
	second.stop()
	second.stop() // Cleanup owns the same idempotent cancellation/join.
	observed, identityErr := (procutil.Source{}).ReadIdentity(second.pid)
	classification := ClassifyEndpointProcess(winner.ProcessIdentity, observed, identityErr)
	if classification != EndpointProcessDead && classification != EndpointProcessReplaced {
		t.Fatalf("joined editor still lacks positive death evidence: %s %v", classification, identityErr)
	}
	if err = owner.Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	after, e := (EndpointStore{Namespace: namespace, Path: filepath.Join(namespace, "state.json"), Stage: filepath.Join(namespace, "stage.json")}).Load()
	if e != nil {
		t.Fatal(e)
	}
	if len(after.Records) != 1 || after.Records[0].Binding.EditorIncarnation != incA || !reflect.DeepEqual(after.Records[0], beforeFirst) {
		t.Fatalf("sweep failed exact-incarnation collection: before %+v after %+v", saved.Records, after.Records)
	}
	liveIdentity, liveErr := (procutil.Source{}).ReadIdentity(first.pid)
	if ClassifyEndpointProcess(beforeFirst.Binding.ProcessIdentity, liveIdentity, liveErr) != EndpointProcessAlive {
		t.Fatalf("first editor was not retained alive: %v", liveErr)
	}
	if p := owner.Lookup(ctx, namespace, docA); p.Code != "ok" || p.EditorEndpoint != nil {
		t.Fatalf("process collection resurrected displaced fallback: %+v", p)
	}

}

func TestNvimPrivateNativeBufferTuple(t *testing.T) {
	poisonNativeParent(t)
	f := newPrivateNativeEditor(t, "")
	ctx := context.Background()
	req := RegisterRequest{Namespace: f.root, DocumentID: docA, EditorIncarnation: incA, AttachID: "1", Buffer: 1, Socket: f.socket}
	for _, buffer := range []int{1, 2} {
		req.Buffer = buffer
		got, err := f.rpc.Describe(ctx, req)
		wantFile := f.file
		if buffer == 2 {
			wantFile += ".second"
		}
		if err != nil || got == nil || got.Buffer != buffer || got.File != wantFile {
			t.Fatalf("native buffer %d: %+v %v", buffer, got, err)
		}
	}
	req.Buffer = 999
	if got, err := f.rpc.Describe(ctx, req); err != nil || got != nil {
		t.Fatalf("invalid real buffer: %+v %v", got, err)
	}
}
