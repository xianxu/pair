package remotesession

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/netip"
	"time"
)

// sessionOperations belongs to one session and is accessed only under gate.mu.
// Cookie retirement releases its replay table; no global historical-session map.
type sessionOperations struct {
	replay Replay
	tokens float64
	refill time.Time
	stream bool
}

type dispatchCompletion struct {
	result DocumentResult
	err    error
}

func (s *Supervisor) dispatchReady(e *remoteEpoch, session sessionRef, now time.Time) bool {
	if s.epoch != e || e == nil || s.gate.state.phase != phaseReady || session.epoch != s.gate.state.epoch || session.operations == nil || !now.Before(session.expires) {
		return false
	}
	select {
	case <-session.done:
		return false
	default:
	}
	return e.ctx.Err() == nil
}

// Dispatch owns completion independently of the HTTP observer. All durable work
// runs outside the lifecycle gate; only nonblocking backend handoff holds it.
func (s *Supervisor) Dispatch(ctx context.Context, e *remoteEpoch, session sessionRef, id AccessIdentity, ip string, request DocumentRequest) (DocumentResult, error) {
	if err := ctx.Err(); err != nil {
		return DocumentResult{Outcome: "refused"}, err
	}
	raw, err := json.Marshal(request)
	if err != nil {
		return DocumentResult{}, err
	}
	request, err = decodeDocumentRequest(raw)
	if err != nil {
		return DocumentResult{}, err
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil || addr.Zone() != "" || !id.validAt(time.Now()) {
		return DocumentResult{}, errSession
	}
	ip = addr.Unmap().String()
	if request.Action == ActionRead {
		return DocumentResult{}, errors.New("use the registered document read route")
	}
	s.gate.mu.Lock()
	if !s.dispatchReady(e, session, time.Now()) || s.backend == nil {
		s.gate.mu.Unlock()
		return DocumentResult{}, errors.New("remote document adapter unavailable or session revoked")
	}
	if s.gate.forgotten[request.Document.ID] || e.operations >= 8 {
		s.gate.mu.Unlock()
		return DocumentResult{}, errors.New("remote document busy or forgotten; inspect before retry")
	}
	admission, err := session.operations.replay.Admit(request.Sequence, raw)
	if err != nil {
		s.gate.mu.Unlock()
		return DocumentResult{}, err
	}
	if admission.kind == replayCached {
		s.gate.mu.Unlock()
		var result DocumentResult
		err = json.Unmarshal(admission.result, &result)
		return result, err
	}
	if admission.kind == replayPending {
		s.gate.mu.Unlock()
		return DocumentResult{Outcome: "pending"}, nil
	}
	now := time.Now()
	op := session.operations
	if op.refill.IsZero() {
		op.tokens = 20
		op.refill = now
	}
	op.tokens += now.Sub(op.refill).Seconds() * 10
	if op.tokens > 20 {
		op.tokens = 20
	}
	op.refill = now
	if op.tokens < 1 {
		result := DocumentResult{Outcome: "refused"}
		encoded, _ := json.Marshal(result)
		_ = op.replay.Complete(request.Sequence, raw, encoded, false)
		s.gate.mu.Unlock()
		return result, errors.New("remote mutation rate reached; wait and inspect")
	}
	op.tokens--
	e.operations++
	if s.documentOperations == nil {
		s.documentOperations = make(map[string]int)
	}
	s.documentOperations[request.Document.ID]++
	e.requests.Add(1)
	s.gate.mu.Unlock()
	done := make(chan dispatchCompletion, 1)
	latest := DocumentResult{Outcome: "uncertain"}
	observe := func(result DocumentResult) { s.gate.mu.Lock(); latest = result; s.gate.mu.Unlock() }
	go func() {
		defer func() {
			s.gate.mu.Lock()
			e.operations--
			s.documentOperations[request.Document.ID]--
			if s.documentOperations[request.Document.ID] == 0 {
				delete(s.documentOperations, request.Document.ID)
			}
			s.gate.mu.Unlock()
			e.requests.Done()
		}()
		done <- s.performDispatch(e, session, id, ip, request, raw, observe)
	}()
	select {
	case completed := <-done:
		return completed.result, completed.err
	case <-ctx.Done():
		select {
		case completed := <-done:
			return completed.result, completed.err
		default:
		}
		s.gate.mu.Lock()
		result := latest
		s.gate.mu.Unlock()
		result.Outcome = "uncertain"
		return result, ctx.Err()
	}
}

func (s *Supervisor) completeDispatch(session sessionRef, request DocumentRequest, raw []byte, result DocumentResult) {
	encoded, err := json.Marshal(result)
	if err != nil || len(encoded) > 1024 {
		result = DocumentResult{Outcome: "uncertain"}
		encoded, _ = json.Marshal(result)
	}
	s.gate.mu.Lock()
	defer s.gate.mu.Unlock()
	_ = session.operations.replay.Complete(request.Sequence, raw, encoded, result.Outcome == "uncertain")
}
func (s *Supervisor) performDispatch(e *remoteEpoch, session sessionRef, id AccessIdentity, ip string, request DocumentRequest, raw []byte, observe func(DocumentResult)) (completed dispatchCompletion) {
	completed.result = DocumentResult{Outcome: "refused"}
	defer func() { s.completeDispatch(session, request, raw, completed.result) }()
	deadline := time.Now().Add(5 * time.Second)
	if session.expires.Before(deadline) {
		deadline = session.expires
	}
	ctx, cancel := context.WithDeadline(e.ctx, deadline)
	defer cancel()
	// The backend owns registry/editor eligibility; Read does not open arbitrary
	// paths. Recheck at its final effect door, not only at this initial snapshot.
	snapshot, err := s.backend.Read(ctx, request.Document)
	if err != nil || snapshot.Ref != request.Document || snapshot.Revision != request.BaseRevision {
		completed.err = errors.New("registered document changed or unavailable")
		return
	}
	if request.Action == ActionNavigate {
		request.destination, err = s.backend.ResolveNavigation(ctx, request.Document, *request.Navigation)
		if err != nil || !request.destination.valid() {
			completed.err = errors.New("remote navigation unavailable")
			return
		}
	}
	if request.Action == ActionEdit || request.Action == ActionSubmit {
		if request.Action == ActionSubmit {
			err = s.origins.MarkSubmitted(request.Document.ID)
		} else {
			err = s.origins.MarkRemote(request.Document.ID)
		}
		if err != nil {
			completed.err = errors.New("remote origin evidence unavailable; inspect storage")
			return
		}
	}
	principal := sha256.Sum256([]byte(id.issuer + "\x00" + id.subject + "\x00" + id.audience + "\x00" + id.email))
	audit := AuditRecord{At: time.Now().UTC(), Action: string(request.Action), Principal: hex.EncodeToString(principal[:]), Session: hex.EncodeToString(session.key[:]), Epoch: session.epoch, DocumentID: request.Document.ID, Sequence: request.Sequence, Before: request.BaseRevision}
	if err = s.audit.AppendIntent(audit); err != nil {
		completed.err = errors.New("remote audit unavailable; inspect storage")
		s.Kill()
		return
	}
	s.gate.mu.Lock()
	var results <-chan DocumentResult
	var accepted bool
	if s.dispatchReady(e, session, time.Now()) && !s.gate.forgotten[request.Document.ID] && !s.gate.forgotten[request.EffectDocument().ID] {
		permit := newPermit(&s.gate, session, request.EffectDocument(), ctx, session.expires, time.Now)
		permit.state.source = DocumentSource{Email: id.email, IP: ip}
		results, accepted = s.backend.Dispatch(ctx, request, permit)
	}
	s.gate.mu.Unlock()
	if accepted {
		completed.result = DocumentResult{Outcome: "uncertain"}
		select {
		case result, ok := <-results:
			if ok {
				completed.result = result
			}
		case <-ctx.Done():
		}
	}
	switch completed.result.Outcome {
	case "applied", "refused", "uncertain":
	default:
		completed.result = DocumentResult{Outcome: "uncertain"}
	}
	encoded, resultErr := json.Marshal(completed.result)
	if resultErr != nil || len(encoded) > 1024 || !auditOpaque(completed.result.Revision, 128) || !auditOpaque(completed.result.ReceiptID, 128) {
		completed.result = DocumentResult{Outcome: "uncertain"}
		completed.err = errors.New("remote result invalid; inspect document and receipts")
		s.Kill()
	}
	observe(completed.result)
	audit.At = time.Now().UTC()
	audit.After = completed.result.Revision
	audit.Outcome = completed.result.Outcome
	audit.ReceiptID = completed.result.ReceiptID
	if err = s.audit.AppendResult(audit); err != nil {
		completed.result.Outcome = "uncertain"
		observe(completed.result)
		completed.err = errors.New("remote result audit uncertain; inspect receipt and document")
		s.Kill()
	}
	return
}

// FenceDocument is local lifecycle authority, not a remote route. It rejects
// new permits immediately; callers retry if previously owned work is unsettled.
// The fence remains until exact-ID retirement completes, never for a reused path.
func (s *Supervisor) FenceDocument(id string) error {
	if !validDocumentID(id) {
		return errors.New("invalid forgotten document ID")
	}
	s.gate.mu.Lock()
	if !s.gate.forgotten[id] && len(s.gate.forgotten) >= maxOriginMarkers {
		s.gate.mu.Unlock()
		return errors.New("forget fence capacity reached; restart Couch and retry")
	}
	if s.gate.forgotten == nil {
		s.gate.forgotten = make(map[string]bool)
	}
	s.gate.forgotten[id] = true
	active := s.documentOperations[id]
	s.gate.mu.Unlock()
	if active != 0 {
		return errors.New("document work pending; inspect and retry forget")
	}
	return nil
}

// CompleteForget releases only ephemeral capacity after the sole local origin
// writer retired the exact absent ID. It neither resets an epoch nor grants a
// session; the backend must still reject that unregistered document identity.
func (s *Supervisor) CompleteForget(id string) error {
	marked, pending, err := s.origins.Inspect(id)
	if err != nil {
		return err
	}
	submitted, err := s.origins.Submitted(id)
	if err != nil {
		return err
	}
	if marked || pending || submitted {
		return errors.New("forget evidence not retired")
	}
	s.gate.mu.Lock()
	defer s.gate.mu.Unlock()
	if s.documentOperations[id] != 0 {
		return errors.New("document work pending")
	}
	delete(s.gate.forgotten, id)
	return nil
}
