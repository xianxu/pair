package drive

import (
	"context"
	"errors"

	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Seat is a viewer's place in a cast: who they are, and the key that
// authenticates their requests. Anyone with the drive link may take a seat
// (the view-only and pointer links offer none, pair#506); driving needs the
// operator's grant, per seat.
type Seat struct {
	Viewer recording.Author
	// Key is secret to the viewer's shell: 32 random bytes, hex.
	Key string
}

// Controller is remote driving's control (#466): viewers' seats, who drives,
// and the driver's input. Every control change is a recording.Ctl record put
// into the cast's record stream. The real one (#467) sits behind
// POST /<token>/drive/{hello,control,input}, and its changes enter the
// recorder's queue, the stream's one writer.
//
// The operator always holds the last word: Revoke takes control back in any
// state, and a seat whose viewer has gone (no stream for the grace period)
// leaves on its own, handing control back. Neither waits on a viewer.
type Controller interface {
	// Drawing is a projection of the driving holder with a sketch-only turn
	// witness. Metadata authentication grants no drawing authority.
	DrawingAuthority() DrawingAuthority
	AdmitDrawing(expectedEpoch uint64, seatKey string) (DrawingPermit, error)
	AdmitOperatorDrawing(expectedEpoch uint64) (DrawingPermit, error)
	AuthenticateDrawingAuthor(seatKey string) (recording.Author, error)
	AdmitDrawingMetadata(seatKey string) (DrawingIdentity, error)
	DriveCapability() capability.Snapshot
	SubscribeDriveCapability() (<-chan capability.Snapshot, func())
	SetDriveAllowed(context.Context, bool) error
	RetryDrive(context.Context) error
	AdmitDrive() (DriveAdmission, error)
	AdmitDriveGeneration(uint64) (DriveAdmission, error)
	// Join gives a new viewer a seat. name is the viewer's own label.
	Join(name string) (Seat, error)
	// Attach marks a seat present for the life of one viewer stream (the
	// events route's ?viewer=). A seat with no attached stream for the
	// grace period leaves (a ctl leave when it drove or asked).
	Attach(viewer string) (detach func(), err error)

	// Request asks to drive; Release gives control back or withdraws a
	// request. Input injects a batch, for the driver only.
	Request(key string) error
	Release(key string) error
	Input(ctx context.Context, key string, b Batch) error

	// Grant lets a viewer who asked drive (a previous driver loses
	// control); Deny turns a request down. Revoke takes control back from
	// whoever drives; with nobody driving it does nothing.
	Grant(viewer string) error
	Deny(viewer string) error
	Revoke() error

	// Control is the current state; Subscribe delivers the latest state,
	// starting with the current one, for Couch's indicator. The channel
	// holds one value and never blocks the controller; it closes at Close.
	Control() recording.Control
	Subscribe() (states <-chan recording.Control, cancel func())
	// Close ends every seat (with the cast). Idempotent.
	Close() error
}

var (
	// ErrNoSeat: an unknown key or viewer (never joined, or left).
	ErrNoSeat = errors.New("drive: no such seat")
	// ErrNotDriver: input from a seat that doesn't drive.
	ErrNotDriver = errors.New("drive: not driving")
	// ErrClosed: the cast ended.
	ErrClosed = errors.New("drive: the cast has ended")
)

// ErrControlOp is recording.ErrControl: a control op the state doesn't allow
// (asking twice, granting a viewer who didn't ask).
var ErrControlOp = recording.ErrControl

// ControlWriter is an ordered control sink with an acknowledgment. Production
// asynchronous writers implement it so the controller observes the committed
// result, not enqueue acceptance. Ordinary recording.Sink fixtures are synchronous.
type ControlWriter interface {
	WriteControl(context.Context, *recording.Ctl) error
}
