package remotesession

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

type fixtureDocumentOperation struct {
	ctx     context.Context
	request DocumentRequest
	permit  Permit
	result  chan DocumentResult
}

// fixtureDocuments is stateful across reads, navigation, held dispatch and editor
// rebinding. It never opens a file/process or stands in for nvim acceptance.
type fixtureDocuments struct {
	mu        sync.Mutex
	documents map[string]DocumentSnapshot
	links     map[string]DocumentRef
	opened    []DocumentRef
	effects   int
	jobs      chan fixtureDocumentOperation
	stop      context.CancelFunc
	done      chan struct{}
	entered   chan struct{}
	release   <-chan struct{}
}

func newFixtureDocuments(t *testing.T, snapshots ...DocumentSnapshot) *fixtureDocuments {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	f := &fixtureDocuments{documents: make(map[string]DocumentSnapshot), links: make(map[string]DocumentRef), jobs: make(chan fixtureDocumentOperation, 1), stop: cancel, done: make(chan struct{}), entered: make(chan struct{}, 8)}
	for _, snapshot := range snapshots {
		f.documents[snapshot.Ref.ID] = snapshot
	}
	go f.work(ctx)
	t.Cleanup(func() { cancel(); <-f.done })
	return f
}
func (f *fixtureDocuments) List(context.Context) ([]DocumentSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]DocumentSummary, 0, len(f.documents))
	for _, s := range f.documents {
		result = append(result, DocumentSummary{Ref: s.Ref, Title: "registered document"})
	}
	return result, nil
}
func (f *fixtureDocuments) Read(ctx context.Context, ref DocumentRef) (DocumentSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return DocumentSnapshot{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	snapshot, ok := f.documents[ref.ID]
	if !ok || snapshot.Ref != ref {
		return DocumentSnapshot{}, errors.New("document unavailable")
	}
	return snapshot, nil
}
func (f *fixtureDocuments) ResolveNavigation(ctx context.Context, source DocumentRef, target NavigationTarget) (DocumentRef, error) {
	if _, err := f.Read(ctx, source); err != nil {
		return DocumentRef{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	destination, ok := f.links[target.Kind+":"+target.Key]
	current, eligible := f.documents[destination.ID]
	if !ok || !eligible || current.Ref != destination {
		return DocumentRef{}, errors.New("navigation unavailable")
	}
	return destination, nil
}
func (f *fixtureDocuments) Dispatch(ctx context.Context, request DocumentRequest, permit Permit) (<-chan DocumentResult, bool) {
	result := make(chan DocumentResult, 1)
	select {
	case <-f.done:
		return nil, false
	default:
	}
	select {
	case f.jobs <- fixtureDocumentOperation{ctx, request, permit, result}:
		return result, true
	default:
		return nil, false
	}
}
func (f *fixtureDocuments) Subscribe(ctx context.Context, ref DocumentRef) (<-chan DocumentSnapshot, error) {
	snapshot, err := f.Read(ctx, ref)
	if err != nil {
		return nil, err
	}
	out := make(chan DocumentSnapshot, 1)
	out <- snapshot
	close(out)
	return out, nil
}
func (f *fixtureDocuments) Join(ctx context.Context) error {
	f.stop()
	select {
	case <-f.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (f *fixtureDocuments) work(ctx context.Context) {
	defer close(f.done)
	for {
		select {
		case <-ctx.Done():
			return
		case op := <-f.jobs:
			select {
			case f.entered <- struct{}{}:
			default:
			}
			if f.release != nil {
				select {
				case <-ctx.Done():
					op.result <- DocumentResult{Outcome: "refused"}
					continue
				case <-op.ctx.Done():
					op.result <- DocumentResult{Outcome: "refused"}
					continue
				case <-f.release:
				}
			}
			f.mu.Lock()
			source, ok := f.documents[op.request.Document.ID]
			result := DocumentResult{Outcome: "refused"}
			target := op.request.EffectDocument()
			destination, eligible := f.documents[target.ID]
			valid := ok && source.Ref == op.request.Document && source.Revision == op.request.BaseRevision && eligible && destination.Ref == target
			text := source.Text
			if edit := op.request.Edit; edit != nil {
				if edit.Start < 0 || edit.End < edit.Start || edit.End > len(text) || !utf8.ValidString(text[:edit.Start]) || !utf8.ValidString(text[edit.End:]) {
					valid = false
				} else {
					text = text[:edit.Start] + edit.Text + text[edit.End:]
				}
			}
			if valid {
				op.permit.EffectDoor(target, func() bool {
					f.effects++
					switch op.request.Action {
					case ActionEdit:
						sum := sha256.Sum256([]byte(text))
						source.Text = text
						source.Revision = hex.EncodeToString(sum[:])
						f.documents[source.Ref.ID] = source
					case ActionNavigate:
						f.opened = append(f.opened, target)
					}
					result = DocumentResult{Outcome: "applied", Revision: source.Revision}
					return true
				})
			}
			f.mu.Unlock()
			op.result <- result
		}
	}
}
func TestRemoteStatefulDocumentRebindAtHandoffHasNoEffect(t *testing.T) {
	ref := DocumentRef{ID: originDocA, Generation: "editor-1"}
	f := newFixtureDocuments(t, DocumentSnapshot{Ref: ref, Revision: "r1", Text: "hello"})
	release := make(chan struct{})
	f.release = release
	gate := &admissionGate{state: remoteState{phase: phaseReady, epoch: 1}}
	permit := newPermit(gate, sessionRef{epoch: 1, done: make(chan struct{})}, ref, context.Background(), time.Now().Add(time.Hour), time.Now)
	request := DocumentRequest{Sequence: 1, Action: ActionEdit, Document: ref, BaseRevision: "r1", Edit: &DocumentEdit{Text: "new "}}
	result, ok := f.Dispatch(context.Background(), request, permit)
	if !ok {
		t.Fatal("queue refused")
	}
	<-f.entered
	f.mu.Lock()
	current := f.documents[ref.ID]
	current.Ref.Generation = "editor-2"
	f.documents[ref.ID] = current
	f.mu.Unlock()
	close(release)
	if got := <-result; got.Outcome != "refused" {
		t.Fatal("rebound effect", got)
	}
	if f.effects != 0 {
		t.Fatal("effect crossed editor generation")
	}
}

func TestRemoteStatefulDocumentKillWhileQueuedHasNoEffect(t *testing.T) {
	ref := DocumentRef{ID: originDocA, Generation: "editor-1"}
	f := newFixtureDocuments(t, DocumentSnapshot{Ref: ref, Revision: "r1", Text: "hello"})
	release := make(chan struct{})
	f.release = release
	gate := &admissionGate{state: remoteState{phase: phaseReady, epoch: 1}}
	permit := newPermit(gate, sessionRef{epoch: 1, done: make(chan struct{})}, ref, context.Background(), time.Now().Add(time.Hour), time.Now)
	result, ok := f.Dispatch(context.Background(), DocumentRequest{Sequence: 1, Action: ActionEdit, Document: ref, BaseRevision: "r1", Edit: &DocumentEdit{Text: "new "}}, permit)
	if !ok {
		t.Fatal("queue refused")
	}
	<-f.entered
	gate.mu.Lock()
	gate.state, _ = ReduceRemote(gate.state, remoteEvent{kind: eventKill})
	gate.mu.Unlock()
	close(release)
	if got := <-result; got.Outcome != "refused" {
		t.Fatal("effect after kill", got)
	}
	if f.effects != 0 {
		t.Fatal("queued effect survived kill")
	}
}
func TestRemoteStatefulNavigationValidatesBeforeOpening(t *testing.T) {
	source := DocumentRef{ID: originDocA, Generation: "editor-1"}
	target := DocumentRef{ID: originDocB, Generation: "editor-2"}
	f := newFixtureDocuments(t, DocumentSnapshot{Ref: source, Revision: "r1", Text: "source"})
	f.links["link:opaque-link"] = target
	if _, err := f.ResolveNavigation(context.Background(), source, NavigationTarget{Kind: "link", Key: "opaque-link"}); err == nil {
		t.Fatal("unregistered target resolved")
	}
	if len(f.opened) != 0 {
		t.Fatal("navigation opened before eligibility")
	}
	f.documents[target.ID] = DocumentSnapshot{Ref: target, Revision: "r2", Text: "target"}
	resolved, err := f.ResolveNavigation(context.Background(), source, NavigationTarget{Kind: "link", Key: "opaque-link"})
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	f.release = release
	gate := &admissionGate{state: remoteState{phase: phaseReady, epoch: 2}}
	permit := newPermit(gate, sessionRef{epoch: 2, done: make(chan struct{})}, resolved, context.Background(), time.Now().Add(time.Hour), time.Now)
	request := DocumentRequest{Sequence: 1, Action: ActionNavigate, Document: source, BaseRevision: "r1", Navigation: &NavigationTarget{Kind: "link", Key: "opaque-link"}, destination: resolved}
	result, ok := f.Dispatch(context.Background(), request, permit)
	if !ok {
		t.Fatal("queue refused")
	}
	<-f.entered
	f.mu.Lock()
	delete(f.documents, target.ID)
	f.mu.Unlock()
	close(release)
	if got := <-result; got.Outcome != "refused" || len(f.opened) != 0 {
		t.Fatal("forgotten target opened", got)
	}
}
