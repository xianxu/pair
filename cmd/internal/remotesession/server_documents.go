package remotesession

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const remoteDocumentFrameBytes = 1 << 20
const remoteDocumentListLimit = 4096 // same bound as the registration owner

// documentRoute deliberately accepts no paths, commands, routing identities or
// caller provenance. #760 consumes this closed API with the existing session.
func (h *RemoteServer) documentRoute(w http.ResponseWriter, r *http.Request, e *remoteEpoch, id AccessIdentity, ip, suffix string) bool {
	route := strings.TrimPrefix(suffix, "v1/")
	if suffix == route {
		return false
	}
	action := RemoteAction("")
	switch route {
	case "documents":
		if r.Method != http.MethodGet {
			return false
		}
	case "read", "stream":
		if r.Method != http.MethodPost {
			return false
		}
		action = ActionRead
	case "edit":
		if r.Method != http.MethodPost {
			return false
		}
		action = ActionEdit
	case "submit":
		if r.Method != http.MethodPost {
			return false
		}
		action = ActionSubmit
	case "navigate":
		if r.Method != http.MethodPost {
			return false
		}
		action = ActionNavigate
	default:
		return false
	}
	cookie, err := r.Cookie("pair-remote")
	if err != nil {
		http.Error(w, "remote handshake required", 401)
		return true
	}
	csrf := r.Header.Values("X-Remote-CSRF")
	if r.Method == http.MethodPost && len(csrf) != 1 {
		http.Error(w, "remote handshake required", 401)
		return true
	}
	session, err := e.sessions.Admit(cookie.Value, r.Header.Get("X-Remote-CSRF"), id, time.Now(), r.Method == http.MethodPost)
	if err != nil {
		http.Error(w, "remote handshake required", 401)
		return true
	}
	h.s.gate.mu.Lock()
	backend := h.s.backend
	ready := h.s.dispatchReady(e, session, time.Now())
	h.s.gate.mu.Unlock()
	if !ready {
		http.Error(w, "remote session revoked", 401)
		return true
	}
	if backend == nil {
		http.Error(w, "remote document adapter unavailable", 503)
		return true
	}
	ctx, closeContext := documentSessionContext(r.Context(), session)
	defer closeContext()
	if route == "documents" {
		list, err := backend.List(ctx)
		if err != nil || len(list) > remoteDocumentListLimit {
			http.Error(w, "registered document list unavailable", 503)
			return true
		}
		h.s.gate.mu.Lock()
		ready = h.s.dispatchReady(e, session, time.Now()) && ctx.Err() == nil
		filtered := make([]DocumentSummary, 0, len(list))
		for _, item := range list {
			if !item.Ref.valid() || !utf8.ValidString(item.Title) || len(item.Title) > remoteDocumentFrameBytes {
				ready = false
				break
			}
			if !h.s.gate.forgotten[item.Ref.ID] {
				filtered = append(filtered, item)
			}
		}
		h.s.gate.mu.Unlock()
		raw, err := boundedDocumentJSON(filtered)
		if !ready || err != nil {
			http.Error(w, "registered document list unavailable", 503)
			return true
		}
		writeDocumentJSON(w, raw)
		return true
	}
	request, err := readDocumentHTTP(r)
	if err != nil || request.Action != action {
		http.Error(w, "remote document request refused", 400)
		return true
	}
	if route == "stream" {
		h.documentStream(w, r.WithContext(ctx), e, session, backend, request.Document)
		return true
	}
	if action == ActionRead {
		snapshot, err := backend.Read(ctx, request.Document)
		if err != nil || snapshot.Ref != request.Document {
			http.Error(w, "registered document changed or unavailable", 409)
			return true
		}
		h.s.gate.mu.Lock()
		ready = h.s.dispatchReady(e, session, time.Now()) && !h.s.gate.forgotten[request.Document.ID] && ctx.Err() == nil
		h.s.gate.mu.Unlock()
		raw, err := boundedDocumentJSON(snapshot)
		if !ready || err != nil {
			http.Error(w, "registered document changed or unavailable", 409)
			return true
		}
		writeDocumentJSON(w, raw)
		return true
	}
	result, err := h.s.Dispatch(ctx, e, session, id, ip, request)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		writeRemoteJSON(w, struct {
			Result DocumentResult
			Error  string
		}{result, "remote operation unconfirmed; inspect document and receipt before retry"})
		return true
	}
	writeRemoteJSON(w, result)
	return true
}

func readDocumentHTTP(r *http.Request) (DocumentRequest, error) {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return DocumentRequest{}, errors.New("JSON required")
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, (64<<10)+1))
	if err != nil {
		return DocumentRequest{}, err
	}
	return decodeDocumentRequest(raw)
}

// No detached watcher survives its handler: cancellation and join are paired.
func documentSessionContext(parent context.Context, session sessionRef) (context.Context, func()) {
	ctx, cancel := context.WithDeadline(parent, session.expires)
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case <-session.done:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, func() { cancel(); <-done }
}

func boundedDocumentJSON(value any) ([]byte, error) {
	switch list := value.(type) {
	case []DocumentSummary:
		var out bytes.Buffer
		out.WriteByte('[')
		for index, item := range list {
			if len(item.Title) > remoteDocumentFrameBytes {
				return nil, errors.New("document title exceeds bound")
			}
			raw, err := json.Marshal(item)
			if err != nil {
				return nil, err
			}
			extra := len(raw)
			if index > 0 {
				extra++
			}
			if out.Len()+extra+2 > remoteDocumentFrameBytes {
				return nil, errors.New("document list exceeds bound")
			}
			if index > 0 {
				out.WriteByte(',')
			}
			out.Write(raw)
		}
		out.WriteByte(']')
		return out.Bytes(), nil
	}
	switch v := value.(type) {
	case DocumentSnapshot:
		if len(v.Text) > remoteDocumentFrameBytes || !utf8.ValidString(v.Text) || !auditOpaque(v.Revision, 128) {
			return nil, errors.New("document snapshot exceeds bound")
		}
	}
	raw, err := json.Marshal(value)
	if err != nil || len(raw)+1 > remoteDocumentFrameBytes {
		return nil, errors.New("document output exceeds bound")
	}
	return raw, nil
}
func writeDocumentJSON(w http.ResponseWriter, raw []byte) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(append(raw, '\n'))
}

func (h *RemoteServer) documentStream(w http.ResponseWriter, r *http.Request, e *remoteEpoch, session sessionRef, backend DocumentBackend, ref DocumentRef) {
	h.s.gate.mu.Lock()
	if !h.s.dispatchReady(e, session, time.Now()) || h.s.gate.forgotten[ref.ID] {
		h.s.gate.mu.Unlock()
		http.Error(w, "registered document unavailable", 409)
		return
	}
	if session.operations.stream {
		h.s.gate.mu.Unlock()
		http.Error(w, "one remote stream per session", 429)
		return
	}
	session.operations.stream = true
	h.s.gate.mu.Unlock()
	defer func() { h.s.gate.mu.Lock(); session.operations.stream = false; h.s.gate.mu.Unlock() }()
	updates, err := backend.Subscribe(r.Context(), ref)
	if err != nil || updates == nil {
		http.Error(w, "registered document stream unavailable", 409)
		return
	}
	controller := http.NewResponseController(w)
	_ = controller.SetReadDeadline(time.Time{})
	w.Header().Set("Content-Type", "application/x-ndjson")
	// The backend owns its bounded subscription queue. This consumer adds no
	// queue or producer goroutine: at most one <=1MiB encoded frame is in flight.
	for {
		select {
		case <-r.Context().Done():
			return
		case snapshot, ok := <-updates:
			if !ok {
				return
			}
			if snapshot.Ref != ref {
				return
			}
			raw, err := boundedDocumentJSON(snapshot)
			if err != nil {
				return
			}
			h.s.gate.mu.Lock()
			ready := h.s.dispatchReady(e, session, time.Now()) && !h.s.gate.forgotten[ref.ID] && r.Context().Err() == nil
			h.s.gate.mu.Unlock()
			if !ready {
				return
			}
			if err := controller.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil && !errors.Is(err, http.ErrNotSupported) {
				return
			}
			if _, err := io.Copy(w, bytes.NewReader(append(raw, '\n'))); err != nil {
				return
			}
			if err := controller.Flush(); err != nil {
				return
			}
		}
	}
}
