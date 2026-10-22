package couchcmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/editorendpoint"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

type editorEndpointService struct {
	namespace string
	owner     *editorendpoint.EndpointOwner
	server    *couchmessage.Server
	cancel    context.CancelFunc
	workers   sync.WaitGroup
	initErr   error
}

func startEditorEndpointService(parent context.Context, namespace string) (*editorEndpointService, error) {
	ctx, cancel := context.WithCancel(parent)
	s := &editorEndpointService{namespace: namespace, cancel: cancel}
	paths, err := artifactpath.EditorEndpoints(namespace)
	if err != nil {
		cancel()
		return nil, err
	}
	resolve := func(ctx context.Context, id string) (string, error) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		rows, err := sessionsync.New(namespace).List()
		if err != nil {
			return "", err
		}
		for _, row := range rows {
			if row.ID == id {
				return row.File, nil
			}
		}
		return "", editorendpoint.ErrDocumentNotRegistered
	}
	random := func() (string, error) {
		var token [16]byte
		if _, err := rand.Read(token[:]); err != nil {
			return "", err
		}
		return hex.EncodeToString(token[:]), nil
	}
	s.owner, s.initErr = editorendpoint.NewEndpointOwner(ctx, editorendpoint.EndpointConfig{Store: editorendpoint.EndpointStore{Namespace: namespace, Path: paths.State(), Stage: paths.Stage()}, RPC: editorendpoint.NvimRPC{}, Processes: procutil.Source{}, Resolve: resolve, Random: random})
	socket, err := couchmessage.SocketPath(namespace, "editor-endpoint")
	if err == nil {
		err = prepareMessageSocket(socket)
	}
	if err == nil {
		s.server, err = couchmessage.StartServer(ctx, socket, s.handle)
	}
	if err != nil {
		s.Close()
		return nil, err
	}
	if s.owner != nil {
		s.workers.Add(1)
		go func() {
			defer s.workers.Done()
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for {
				_ = s.owner.Sweep(ctx)
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
	}
	return s, nil
}

func (s *editorEndpointService) Close() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.owner != nil {
		_ = s.owner.Close()
	}
	if s.server != nil {
		_ = s.server.Close()
	}
	s.workers.Wait()
}

func (s *editorEndpointService) handle(ctx context.Context, raw []byte) ([]byte, error) {
	reply := editorendpoint.EndpointReply{Version: 1, Code: "invalid", Error: "invalid editor endpoint request"}
	finish := func() ([]byte, error) {
		if reply.Code != "ok" && reply.Error == "" {
			reply.Error = endpointRecovery(reply.Code)
		}
		out, err := json.Marshal(reply)
		if err != nil || len(out) > 1<<20 {
			return json.Marshal(editorendpoint.EndpointReply{Version: 1, Code: "unavailable", Error: "editor endpoint reply exceeds bound"})
		}
		return out, nil
	}
	if len(raw) > (256<<10)+4096 {
		return finish()
	}
	var request endpointIPCRequest
	owned, err := endpointDecodeObject(raw, &request, map[string]bool{"Register": true, "Unregister": true, "DocumentID": true}, nil, false)
	if err != nil {
		return finish()
	}
	if request.Namespace != s.namespace {
		return finish()
	}
	switch request.Operation {
	case "register":
		if request.Register == nil || request.Unregister != nil || request.DocumentID != "" || request.Register.Namespace != s.namespace || len(owned["Register"]) > 8192 || endpointDecodeRequest(owned["Register"], request.Register) != nil {
			return finish()
		}
	case "unregister":
		if request.Unregister == nil || request.Register != nil || request.DocumentID != "" || request.Unregister.Namespace != s.namespace || len(owned["Unregister"]) > 256<<10 || endpointDecodeRequest(owned["Unregister"], request.Unregister) != nil {
			return finish()
		}
	case "lookup":
		if request.Register != nil || request.Unregister != nil || !editorEndpointDocumentID(request.DocumentID) {
			return finish()
		}
	default:
		return finish()
	}
	if s.owner == nil || s.initErr != nil {
		reply.Code = "unavailable"
		reply.Error = "editor endpoint storage unavailable; inspect storage and restart Couch"
		return finish()
	}
	switch request.Operation {
	case "register":
		reply = s.owner.Register(ctx, *request.Register)
	case "unregister":
		reply = s.owner.Unregister(ctx, *request.Unregister)
	case "lookup":
		reply = s.owner.Lookup(ctx, request.Namespace, request.DocumentID)
	}
	return finish()
}

func editorEndpointDocumentID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, c := range id {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func endpointRecovery(code string) string {
	switch code {
	case "superseded", "stale":
		return "editor attachment superseded; explicitly reopen to attach again"
	case "desktop-active":
		return "a desktop editor is active; use that editor"
	case "conflict":
		return "attachment payload conflicts; inspect and explicitly reopen"
	case "capacity":
		return "endpoint capacity reached; close unused editors and retry"
	case "not-registered":
		return "document is not registered; register it before attaching"
	case "uncertain":
		return "endpoint publication uncertain; retry the identical intent or inspect and restart Couch"
	default:
		return "editor endpoint unavailable; inspect and retry"
	}
}
