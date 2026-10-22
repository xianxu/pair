package drive

import (
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// DrawingAuthority projects the current driving turn. Epoch is sketch-only:
// it does not change ordinary driving capability or input admission semantics.
// A client captures Epoch when creating work, including before ingress; a
// stale A request cannot acquire a later A turn after a handoff through B.
type DrawingAuthority struct {
	Epoch  uint64 `json:"epoch"`
	Holder string `json:"holder"`
	Ready  bool   `json:"ready"`
}

// DrawingPermit is an opaque witness to one driving turn. It is safe to check
// from the recorder actor without taking controller locks. Annotation/session
// proof and the recorder's actual holder remain separate final checks.
type DrawingPermit struct {
	principal recording.Author
	revoked   <-chan struct{}
}

func (p DrawingPermit) Principal() recording.Author { return p.principal }
func (p DrawingPermit) Check() error {
	if p.revoked == nil {
		return ErrDriveUnavailable
	}
	select {
	case <-p.revoked:
		return ErrDriveUnavailable
	default:
		return nil
	}
}

func (f *MemController) DrawingAuthority() DrawingAuthority {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.drawingAuthorityLocked()
}

func (f *MemController) drawingAuthorityLocked() DrawingAuthority {
	holder := f.control.Driving()
	if holder == "" {
		holder = recording.OperatorID
	}
	return DrawingAuthority{Epoch: f.drawingEpoch, Holder: holder, Ready: !f.closed && f.drawingWitness != nil && (f.authority.Phase == capability.On || f.authority.Phase == capability.Off)}
}

func (f *MemController) AdmitDrawing(expectedEpoch uint64, seatKey string) (DrawingPermit, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return DrawingPermit{}, ErrClosed
	}
	s, err := f.bySeat(seatKey)
	if err != nil {
		return DrawingPermit{}, err
	}
	a := f.drawingAuthorityLocked()
	if !a.Ready || expectedEpoch != a.Epoch {
		return DrawingPermit{}, ErrDriveUnavailable
	}
	if f.authority.Phase != capability.On || a.Holder != s.Viewer.ID {
		return DrawingPermit{}, ErrNotDriver
	}
	return DrawingPermit{principal: s.Viewer, revoked: f.drawingWitness}, nil
}

func (f *MemController) AdmitOperatorDrawing(expectedEpoch uint64) (DrawingPermit, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return DrawingPermit{}, ErrClosed
	}
	a := f.drawingAuthorityLocked()
	if !a.Ready || expectedEpoch != a.Epoch {
		return DrawingPermit{}, ErrDriveUnavailable
	}
	if a.Holder != recording.OperatorID {
		return DrawingPermit{}, ErrNotDriver
	}
	return DrawingPermit{principal: recording.Author{ID: recording.OperatorID, Name: "Operator"}, revoked: f.drawingWitness}, nil
}

// AuthenticateDrawingAuthor authenticates a surviving seat for metadata-only
// label/status work after handoff. It grants no drawing permission or new turn.
func (f *MemController) AuthenticateDrawingAuthor(seatKey string) (recording.Author, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return recording.Author{}, ErrClosed
	}
	s, err := f.bySeat(seatKey)
	if err != nil {
		return recording.Author{}, err
	}
	return s.Viewer, nil
}

// DrawingIdentity proves only the original seat's lifetime, including after
// handoff. Its final actor check takes no controller lock.
type DrawingIdentity struct{ permit DrawingPermit }

func (p DrawingIdentity) Principal() recording.Author { return p.permit.Principal() }
func (p DrawingIdentity) Check() error                { return p.permit.Check() }
func (f *MemController) AdmitDrawingMetadata(key string) (DrawingIdentity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return DrawingIdentity{}, ErrClosed
	}
	s, err := f.bySeat(key)
	if err != nil {
		return DrawingIdentity{}, err
	}
	return DrawingIdentity{DrawingPermit{principal: s.Viewer, revoked: s.drawingLife}}, nil
}

// beginDrawingTransitionLocked fences before any ordered ctl/reset effect.
// Exhaustion never wraps into an epoch a client might have retained.
func (f *MemController) beginDrawingTransitionLocked() {
	if f.drawingWitness != nil {
		close(f.drawingWitness)
		f.drawingWitness = nil
	}
	if f.drawingEpoch < recording.MaxSketchInteger {
		f.drawingEpoch++
	}
	f.publishCapabilityLocked()
}

func (f *MemController) renewDrawingLocked() {
	if f.closed || f.drawingEpoch >= recording.MaxSketchInteger || f.drawingWitness != nil || (f.authority.Phase != capability.On && f.authority.Phase != capability.Off) {
		return
	}
	f.drawingWitness = make(chan struct{})
	f.publishCapabilityLocked()
}

func (f *MemController) freezeDrawingLocked() {
	if f.drawingWitness != nil {
		f.beginDrawingTransitionLocked()
	}
}
