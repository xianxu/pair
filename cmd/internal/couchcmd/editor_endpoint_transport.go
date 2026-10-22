package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"path/filepath"
	"reflect"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/editorendpoint"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type endpointIPCRequest struct {
	Operation  string                            `json:"Operation"`
	Namespace  string                            `json:"Namespace,omitempty"`
	DocumentID string                            `json:"DocumentID,omitempty"`
	Register   *editorendpoint.RegisterRequest   `json:"Register,omitempty"`
	Unregister *editorendpoint.UnregisterRequest `json:"Unregister,omitempty"`
}

// endpointDecodeObject enforces exact owned names even though encoding/json
// ordinarily accepts case-insensitive aliases. Replies tolerate additive fields;
// requests refuse them before crossing the authority boundary.
func endpointDecodeObject(raw []byte, target any, optional, nullable map[string]bool, additive bool) (map[string]json.RawMessage, error) {
	known := strictjson.KnownFields(reflect.TypeOf(target).Elem())
	owned, unknown, err := strictjson.SplitObject(raw, known, nullable)
	if err != nil {
		return nil, err
	}
	if !additive && len(unknown) > 0 {
		return nil, errors.New("unknown endpoint field")
	}
	for key := range known {
		if _, ok := owned[key]; !ok && !optional[key] {
			return nil, errors.New("missing endpoint field")
		}
	}
	body, err := strictjson.EncodeObject(owned, nil, known)
	if err != nil {
		return nil, err
	}
	if err = strictjson.Decode(body, target); err != nil {
		return nil, err
	}
	return owned, nil
}

func endpointDecodeRequest(raw []byte, target any) error {
	optional := map[string]bool{"ProcessIdentity": true}
	owned, err := endpointDecodeObject(raw, target, optional, nil, false)
	if err != nil {
		return err
	}
	if _, ok := target.(*editorendpoint.UnregisterRequest); ok {
		var entries []json.RawMessage
		if err = json.Unmarshal(owned["Entries"], &entries); err != nil {
			return err
		}
		for _, rawEntry := range entries {
			var entry editorendpoint.UnregisterEntry
			if _, err = endpointDecodeObject(rawEntry, &entry, nil, nil, false); err != nil {
				return err
			}
		}
	}
	return editorendpoint.ValidateEndpointRequest(target)
}

func endpointReplyExit(code string) (int, bool) {
	switch code {
	case "ok":
		return 0, true
	case "invalid":
		return 2, true
	case "unavailable", "capacity":
		return 3, true
	case "superseded", "stale", "conflict", "not-registered", "desktop-active":
		return 4, true
	case "uncertain":
		return 5, true
	default:
		return 5, false
	}
}

func decodeEndpointCLIReply(raw []byte) (editorendpoint.EndpointReply, error) {
	var reply editorendpoint.EndpointReply
	if len(raw) > 1<<20 {
		return reply, errors.New("endpoint response exceeds limit")
	}
	owned, err := endpointDecodeObject(raw, &reply, map[string]bool{"Changed": true, "Results": true}, map[string]bool{"EditorEndpoint": true, "Results": true}, true)
	if err != nil {
		return reply, err
	}
	if _, ok := endpointReplyExit(reply.Code); !ok || reply.Version != 1 || reply.Code == "ok" && reply.Error != "" {
		return reply, errors.New("invalid endpoint response")
	}
	if reply.EditorEndpoint != nil {
		var binding editorendpoint.EndpointBinding
		if _, err = endpointDecodeObject(owned["EditorEndpoint"], &binding, nil, nil, true); err != nil {
			return reply, err
		}
		// The authority's own codec validates the complete binding, including native
		// birth, generation and file, without duplicating those model rules here.
		_, err = editorendpoint.EncodeEndpointState(editorendpoint.EndpointState{Version: 1, Namespace: binding.Namespace, Records: []editorendpoint.EndpointRecord{{Binding: binding, Status: "current"}}})
		if err != nil {
			return reply, err
		}
	}
	if rawResults, ok := owned["Results"]; ok && !bytes.Equal(bytes.TrimSpace(rawResults), []byte("null")) {
		var items []json.RawMessage
		if err = json.Unmarshal(rawResults, &items); err != nil {
			return reply, err
		}
		if len(items) > 256 {
			return reply, errors.New("too many endpoint results")
		}
		for _, rawResult := range items {
			var result editorendpoint.UnregisterResult
			if _, err = endpointDecodeObject(rawResult, &result, nil, nil, true); err != nil {
				return reply, err
			}
			switch result.Status {
			case "removed", "absent", "stale":
			default:
				return reply, errors.New("invalid endpoint result")
			}
		}
	}
	encoded, err := json.Marshal(reply)
	if err != nil || len(encoded)+1 > 1<<20 {
		return reply, errors.New("encoded endpoint response exceeds limit")
	}
	return reply, nil
}

func runEditorEndpointCLI(inv cliInvocation, rt Runtime, stdin io.Reader, stdout, stderr io.Writer, call func(context.Context, string, any, any) error) int {
	// Every parsed operation emits one typed envelope, including local refusals.
	emit := func(reply editorendpoint.EndpointReply) int {
		_ = json.NewEncoder(stdout).Encode(reply)
		code, _ := endpointReplyExit(reply.Code)
		return code
	}
	fail := func(code, message string) int {
		return emit(editorendpoint.EndpointReply{Version: 1, Code: code, Error: message})
	}
	_ = stderr
	if len(inv.args) == 0 {
		return fail("invalid", "invalid editor endpoint operation")
	}
	req := endpointIPCRequest{Operation: inv.args[0]}
	switch req.Operation {
	case "register", "unregister":
		if len(inv.args) != 1 {
			return fail("invalid", "invalid editor endpoint operation")
		}
		limit := 8192
		if req.Operation == "unregister" {
			limit = 256 << 10
		}
		raw, err := io.ReadAll(io.LimitReader(stdin, int64(limit+1)))
		if err != nil || len(raw) > limit {
			return fail("invalid", "invalid or oversized endpoint request")
		}
		if req.Operation == "register" {
			req.Register = &editorendpoint.RegisterRequest{}
			if endpointDecodeRequest(raw, req.Register) != nil {
				return fail("invalid", "invalid endpoint request")
			}
			req.Namespace = req.Register.Namespace
		} else {
			req.Unregister = &editorendpoint.UnregisterRequest{}
			if endpointDecodeRequest(raw, req.Unregister) != nil {
				return fail("invalid", "invalid endpoint request")
			}
			req.Namespace = req.Unregister.Namespace
		}
	case "lookup":
		if len(inv.args) != 3 {
			return fail("invalid", "invalid endpoint lookup")
		}
		req.Namespace, req.DocumentID = inv.args[1], inv.args[2]
		if !editorEndpointDocumentID(req.DocumentID) || !filepath.IsAbs(req.Namespace) || filepath.Clean(req.Namespace) != req.Namespace {
			return fail("invalid", "invalid endpoint lookup")
		}
	default:
		return fail("invalid", "invalid editor endpoint operation")
	}
	ns, err := rt.ResolveNamespace()
	if err != nil {
		return fail("unavailable", "endpoint namespace unavailable")
	}
	if !filepath.IsAbs(req.Namespace) || filepath.Clean(req.Namespace) != req.Namespace || req.Namespace != ns.Dir() {
		return fail("invalid", "endpoint namespace does not match Couch")
	}
	socket, err := couchmessage.SocketPath(ns.Dir(), "editor-endpoint")
	if err != nil {
		return fail("unavailable", "endpoint authority unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var raw json.RawMessage
	if err = call(ctx, socket, req, &raw); err != nil {
		var op *net.OpError
		if errors.As(err, &op) && op.Op == "dial" {
			return fail("unavailable", "endpoint authority unavailable")
		}
		return fail("uncertain", "endpoint outcome uncertain; retry identical intent")
	}
	reply, err := decodeEndpointCLIReply(raw)
	if err != nil {
		return fail("uncertain", "endpoint outcome uncertain; retry identical intent")
	}
	if b := reply.EditorEndpoint; b != nil {
		document := req.DocumentID
		if req.Register != nil {
			document = req.Register.DocumentID
		}
		if b.Namespace != req.Namespace || (document != "" && b.DocumentID != document) {
			return fail("uncertain", "endpoint outcome uncertain; retry identical intent")
		}
	}
	return emit(reply)
}
