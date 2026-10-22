package editorendpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// EditorRPC distinguishes positive absence (nil, nil) from unavailable evidence.
type EditorRPC interface {
	Describe(context.Context, RegisterRequest) (*EditorDescription, error)
	Superseded(context.Context, EndpointBinding) error
}
type EditorDescription = EndpointBinding

// NvimRPC uses Neovim's installed native RPC client. IO is the owned, bounded
// foreground process seam shared with provisioning; it never invokes a shell.
// The execution budget is one second. Cancellation preserves the runner's
// existing two-second process-group grace and joins teardown before returning.
// Parent output bounds do not impose a heap limit on the native Neovim client.
type NvimRPC struct {
	IO      couchcore.ProvisionIO
	Program string
}

const endpointLua = `(function() local value = require("parley.remote_session").endpoint(_A); if value == nil then return vim.NIL end; return value end)()`
const supersededLua = `(function() local value = require("parley.remote_session").superseded(_A); if value == nil then return vim.NIL end; return value end)()`
const maxNvimOutput = 1 << 20

type endpointQuery struct {
	Namespace, DocumentID, EditorIncarnation, AttachID string
	Buffer                                             int
}
type supersededNotice struct{ Namespace, DocumentID, EditorIncarnation, AttachID, BindingGeneration string }
type endpointDescriptionWire struct {
	Namespace, DocumentID, EditorIncarnation, AttachID string
	PID                                                int
	Socket                                             string
	Buffer                                             int
	Mode, File                                         string
}

func (n NvimRPC) Describe(ctx context.Context, r RegisterRequest) (*EditorDescription, error) {
	raw, err := n.call(ctx, r.Socket, endpointLua, endpointQuery{r.Namespace, r.DocumentID, r.EditorIncarnation, r.AttachID, r.Buffer})
	if err != nil {
		return nil, err
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	known := map[string]bool{"Namespace": true, "DocumentID": true, "EditorIncarnation": true, "AttachID": true, "PID": true, "Socket": true, "Buffer": true, "Mode": true, "File": true}
	owned, unknown, err := strictjson.SplitObject(raw, known, nil)
	if err != nil || len(unknown) != 0 {
		return nil, errors.New("invalid endpoint description")
	}
	for key := range known {
		if key != "Mode" {
			if _, ok := owned[key]; !ok {
				return nil, errors.New("invalid endpoint description")
			}
		}
	}
	var d endpointDescriptionWire
	if err = strictjson.Decode(raw, &d); err != nil {
		return nil, errors.New("invalid endpoint description")
	}
	if d.Namespace != r.Namespace || d.DocumentID != r.DocumentID || d.EditorIncarnation != r.EditorIncarnation || d.AttachID != r.AttachID || d.Buffer != r.Buffer || d.Socket != r.Socket || d.PID <= 0 || d.File == "" {
		return nil, errors.New("endpoint description is incomplete or mismatched")
	}
	return &EditorDescription{Namespace: d.Namespace, DocumentID: d.DocumentID, EditorIncarnation: d.EditorIncarnation, AttachID: d.AttachID, PID: d.PID, Socket: d.Socket, Buffer: d.Buffer, Mode: r.Mode, File: d.File}, nil
}
func (n NvimRPC) Superseded(ctx context.Context, b EndpointBinding) error {
	raw, err := n.call(ctx, b.Socket, supersededLua, supersededNotice{b.Namespace, b.DocumentID, b.EditorIncarnation, b.AttachID, b.BindingGeneration})
	if err != nil {
		return err
	}
	var value any
	if err = strictjson.Decode(raw, &value); err != nil {
		return errors.New("invalid superseded response")
	}
	return nil
}
func (n NvimRPC) call(ctx context.Context, socket, code string, data any) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	// Vimscript single-quoted strings escape only apostrophes by doubling them.
	// Lua is fixed; all endpoint data crosses json_decode as a separate argument.
	expression := `json_encode(luaeval('` + code + `', json_decode('` + strings.ReplaceAll(string(encoded), "'", "''") + `')))`
	runner := n.IO
	if runner == nil {
		runner = couchcore.OSProvisionIO{}
	}
	program := n.Program
	if program == "" {
		program = "nvim"
	}
	diagnostics := &endpointDiagnostics{cancel: cancel}
	out, err := runner.Run(ctx, couchcore.ProvisionCommand{Program: program, Args: []string{"--server", socket, "--remote-expr", expression}, Timeout: time.Second, Progress: diagnostics})
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("editor RPC unavailable")
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if len(out) > maxNvimOutput {
		return nil, errors.New("editor RPC stdout exceeds 1 MiB")
	}
	if diagnostics.written > 0 {
		return nil, errors.New("editor RPC reported diagnostics")
	}
	return out, nil
}

// OSProvisionIO retains a bounded diagnostic tail. This sink counts its stderr
// stream and cancels on overflow, without retaining a second copy.
type endpointDiagnostics struct {
	mu      sync.Mutex
	written int
	cancel  context.CancelFunc
}

func (w *endpointDiagnostics) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.written += len(p)
	if w.written > maxNvimOutput {
		w.cancel()
		return 0, errors.New("editor RPC stderr exceeds 1 MiB")
	}
	return len(p), nil
}
