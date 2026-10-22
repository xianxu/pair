package editorendpoint

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

type endpointCommandIO struct {
	run func(context.Context, couchcore.ProvisionCommand) ([]byte, error)
}

func (f endpointCommandIO) Run(ctx context.Context, r couchcore.ProvisionCommand) ([]byte, error) {
	return f.run(ctx, r)
}
func TestNvimDescribeFixedCallAndNil(t *testing.T) {
	req := RegisterRequest{Namespace: "/project/$()'\n雪", DocumentID: "doc", EditorIncarnation: "inc", AttachID: "42", Buffer: 7, Socket: "/socket '雪"}
	rpc := NvimRPC{IO: endpointCommandIO{run: func(ctx context.Context, r couchcore.ProvisionCommand) ([]byte, error) {
		if r.Program != "nvim" || r.Timeout != time.Second || len(r.Args) != 4 || r.Args[0] != "--server" || r.Args[1] != req.Socket || r.Args[2] != "--remote-expr" {
			t.Fatalf("command %+v", r)
		}
		tuple := struct {
			Namespace, DocumentID, EditorIncarnation, AttachID string
			Buffer                                             int
		}{req.Namespace, req.DocumentID, req.EditorIncarnation, req.AttachID, req.Buffer}
		b, _ := json.Marshal(tuple)
		want := `json_encode(luaeval('` + endpointLua + `', json_decode('` + strings.ReplaceAll(string(b), "'", "''") + `')))`
		if r.Args[3] != want {
			t.Fatalf("expression\n%s\nwant\n%s", r.Args[3], want)
		}
		return []byte("null\n"), nil
	}}}
	d, err := rpc.Describe(context.Background(), req)
	if err != nil || d != nil {
		t.Fatalf("positive nil: %+v %v", d, err)
	}
}
func TestNvimDescribeErrorsStayUnknown(t *testing.T) {
	for _, tt := range []struct {
		name, stdout, stderr string
		err                  error
	}{{"command", "null", "", errors.New("rpc failed")}, {"diagnostics", "null", "E5108", nil}, {"invalid", "0", "", nil}, {"trailing", "null null", "", nil}, {"oversize", strings.Repeat(" ", 1<<20) + "null", "", nil}} {
		t.Run(tt.name, func(t *testing.T) {
			rpc := NvimRPC{IO: endpointCommandIO{run: func(ctx context.Context, r couchcore.ProvisionCommand) ([]byte, error) {
				if tt.stderr != "" {
					_, _ = r.Progress.Write([]byte(tt.stderr))
				}
				return []byte(tt.stdout), tt.err
			}}}
			if _, err := rpc.Describe(context.Background(), RegisterRequest{}); err == nil {
				t.Fatal("unknown became absent")
			}
		})
	}
}
func TestNvimDescribeTypedReplyAndSuperseded(t *testing.T) {
	req := RegisterRequest{Namespace: "/project", DocumentID: "doc", EditorIncarnation: "inc", AttachID: "42", Buffer: 7, Socket: "/socket", Mode: "desktop"}
	binding := EndpointBinding{Namespace: req.Namespace, DocumentID: req.DocumentID, EditorIncarnation: req.EditorIncarnation, AttachID: req.AttachID, Buffer: req.Buffer, Socket: req.Socket, PID: 42, Mode: "desktop", File: "/file", BindingGeneration: "generation"}
	calls := 0
	rpc := NvimRPC{IO: endpointCommandIO{run: func(ctx context.Context, r couchcore.ProvisionCommand) ([]byte, error) {
		calls++
		if calls == 1 {
			return json.Marshal(map[string]any{"Namespace": binding.Namespace, "DocumentID": binding.DocumentID, "EditorIncarnation": binding.EditorIncarnation, "AttachID": binding.AttachID, "Buffer": binding.Buffer, "Socket": binding.Socket, "PID": binding.PID, "Mode": binding.Mode, "File": binding.File})
		}
		if !strings.Contains(r.Args[3], ".superseded(_A)") || !strings.Contains(r.Args[3], `"BindingGeneration":"generation"`) {
			t.Fatalf("notification %s", r.Args[3])
		}
		return []byte("null"), nil
	}}}
	got, err := rpc.Describe(context.Background(), req)
	binding.BindingGeneration = ""
	if err != nil || !reflect.DeepEqual(got, &binding) {
		t.Fatalf("reply %+v %v", got, err)
	}
	binding.BindingGeneration = "generation"
	if err = rpc.Superseded(context.Background(), binding); err != nil {
		t.Fatal(err)
	}
}

func TestNvimDiagnosticsDoNotEscape(t *testing.T) {
	rpc := NvimRPC{IO: endpointCommandIO{run: func(ctx context.Context, r couchcore.ProvisionCommand) ([]byte, error) {
		_, _ = r.Progress.Write([]byte("private editor text"))
		return nil, errors.New("private editor text")
	}}}
	_, err := rpc.Describe(context.Background(), RegisterRequest{})
	if err == nil || strings.Contains(err.Error(), "private editor text") {
		t.Fatalf("unsafe diagnostics: %v", err)
	}
}
func TestNvimCancellationAndStderrCap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rpc := NvimRPC{IO: endpointCommandIO{run: func(context.Context, couchcore.ProvisionCommand) ([]byte, error) {
		t.Fatal("started canceled call")
		return nil, nil
	}}}
	if _, err := rpc.Describe(ctx, RegisterRequest{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	rpc.IO = endpointCommandIO{run: func(ctx context.Context, r couchcore.ProvisionCommand) ([]byte, error) {
		if _, err := r.Progress.Write(make([]byte, maxNvimOutput+1)); err == nil {
			t.Error("stderr overflow accepted")
		}
		if ctx.Err() == nil {
			t.Error("stderr overflow did not cancel owned process")
		}
		return []byte("null"), nil
	}}
	if _, err := rpc.Describe(context.Background(), RegisterRequest{}); err == nil {
		t.Fatal("stderr overflow became absent")
	}
}

func TestNvimDescriptionModeComesFromRequest(t *testing.T) {
	req := RegisterRequest{Namespace: "/project", DocumentID: "doc", EditorIncarnation: "inc", AttachID: "42", Buffer: 7, Socket: "/socket", Mode: "desktop"}
	rpc := NvimRPC{IO: endpointCommandIO{run: func(context.Context, couchcore.ProvisionCommand) ([]byte, error) {
		return []byte(`{"Namespace":"/project","DocumentID":"doc","EditorIncarnation":"inc","AttachID":"42","Buffer":7,"Socket":"/socket","PID":42,"File":"/file"}`), nil
	}}}
	got, err := rpc.Describe(context.Background(), req)
	if err != nil || got == nil || got.Mode != req.Mode {
		t.Fatalf("pinned response without Mode: %+v %v", got, err)
	}
}
func TestNvimDescriptionRejectsAmbiguousJSONWithSafeErrors(t *testing.T) {
	req := RegisterRequest{Namespace: "/project", DocumentID: "doc", EditorIncarnation: "inc", AttachID: "42", Buffer: 7, Socket: "/socket", Mode: "desktop"}
	body := `{"Namespace":"/project","DocumentID":"doc","EditorIncarnation":"inc","AttachID":"42","Buffer":7,"Socket":"/socket","PID":42,"Mode":"desktop","File":"/file"}`
	cases := map[string]string{
		"duplicate":             strings.Replace(body, `"PID":42`, `"PID":42,"PID":42`, 1),
		"wrong case":            strings.Replace(body, `"Namespace"`, `"namespace"`, 1),
		"unknown private key":   strings.Replace(body, `"PID":42`, `"PID":42,"private secret":true`, 1),
		"private type mismatch": strings.Replace(body, `"PID":42`, `"PID":{"private secret":true}`, 1),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			rpc := NvimRPC{IO: endpointCommandIO{run: func(context.Context, couchcore.ProvisionCommand) ([]byte, error) { return []byte(raw), nil }}}
			_, err := rpc.Describe(context.Background(), req)
			if err == nil || strings.Contains(err.Error(), "private secret") {
				t.Fatalf("ambiguous or unsafe response error: %v", err)
			}
		})
	}
	rpc := NvimRPC{IO: endpointCommandIO{run: func(context.Context, couchcore.ProvisionCommand) ([]byte, error) {
		return []byte(`{"private secret":1,"private secret":2}`), nil
	}}}
	if err := rpc.Superseded(context.Background(), EndpointBinding{}); err == nil || strings.Contains(err.Error(), "private secret") {
		t.Fatalf("unsafe superseded response: %v", err)
	}
}
