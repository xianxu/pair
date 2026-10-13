package drive

import (
	"errors"
	"math"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Limits of one input request from a viewer's shell.
const (
	MaxBody = 8 << 10
	MaxActs = 64
	// MaxText bounds a text or composition act, in runes.
	MaxText = 1000
	// maxCoord bounds page coordinates (CSS px); far past any real page.
	maxCoord = 1 << 20
)

// Act kinds. Pointer acts are in top-page CSS px, as every coordinate of
// the format; the injector maps them to the tab's viewport.
const (
	// ActDown and ActUp press and release a button; ActMove moves the
	// pointer (Button set while dragging).
	ActDown = "down"
	ActUp   = "up"
	ActMove = "move"
	// ActClick is a press and release; Count 2 is a double click.
	ActClick = "click"
	// ActWheel scrolls by DX, DY at X, Y.
	ActWheel = "wheel"
	// ActKey is a key going down or up, to the focused element.
	ActKey = "key"
	// ActText inserts text at the focus (typing, as values: no key events).
	ActText = "text"
	// ActIME sets the IME composition V with selection Sel; the committed
	// result follows as an ActText.
	ActIME = "ime"
)

// Key directions, buttons and modifiers.
const (
	KeyDown    = "down"
	KeyUp      = "up"
	ButtonLeft = "left"
	ModShift   = "shift"
	ModCtrl    = "ctrl"
	ModAlt     = "alt"
	ModMeta    = "meta"
)

// Act is one viewer action. ID is the element the viewer's shell hit at X,
// Y (its nodeAt), for down, click and wheel; Injector.Inject states when the
// tab refuses an act for it (ErrStale).
type Act struct {
	A      string           `json:"a"`
	ID     recording.NodeID `json:"id,omitempty"`
	X      float64          `json:"x,omitempty"`
	Y      float64          `json:"y,omitempty"`
	Button string           `json:"button,omitempty"`
	Count  int              `json:"count,omitempty"`
	DX     float64          `json:"dx,omitempty"`
	DY     float64          `json:"dy,omitempty"`
	Type   string           `json:"type,omitempty"`
	Key    string           `json:"key,omitempty"`
	Code   string           `json:"code,omitempty"`
	Mods   []string         `json:"mods,omitempty"`
	V      string           `json:"v,omitempty"`
	Sel    *[2]int          `json:"sel,omitempty"`
}

// Batch is one input request: the acts in order, and the segment the
// viewer was showing (a batch for a document that has gone is stale).
type Batch struct {
	// Admit is an internal dispatch fence; it is never decoded from viewer input.
	Admit func() bool `json:"-"`
	Seg   int         `json:"seg"`
	Acts  []Act       `json:"acts"`
}

// Parse errors: fixed values that never echo the input.
var (
	ErrBatchShape = errors.New("drive: malformed input batch")
	ErrBatchLimit = errors.New("drive: input batch over its limits")
)

// ParseBatch reads a request body strictly: one object with only these
// fields, 1..MaxActs acts, each well formed for its kind. Buttons default to
// left and a click's count to 1. It checks shape; Confine checks what is
// allowed.
func ParseBatch(body []byte) (Batch, error) {
	if len(body) > MaxBody {
		return Batch{}, ErrBatchLimit
	}
	var b Batch
	if !broadcast.DecodeStrict(body, &b) || b.Seg <= 0 || len(b.Acts) == 0 {
		return Batch{}, ErrBatchShape
	}
	if len(b.Acts) > MaxActs {
		return Batch{}, ErrBatchLimit
	}
	for i := range b.Acts {
		if err := b.Acts[i].normalize(); err != nil {
			return Batch{}, err
		}
	}
	return b, nil
}

func (a *Act) normalize() error {
	for _, m := range a.Mods {
		if m != ModShift && m != ModCtrl && m != ModAlt && m != ModMeta {
			return ErrBatchShape
		}
	}
	switch a.A {
	case ActDown, ActUp, ActMove, ActClick, ActWheel:
		if !coord(a.X) || !coord(a.Y) || !finite(a.DX) || !finite(a.DY) {
			return ErrBatchShape
		}
		if a.A == ActDown || a.A == ActClick || a.A == ActWheel {
			if a.ID <= 0 {
				return ErrBatchShape
			}
		}
		if a.Button == "" && a.A != ActMove && a.A != ActWheel {
			a.Button = ButtonLeft
		}
		if a.A == ActClick {
			if a.Count == 0 {
				a.Count = 1
			}
			if a.Count > 2 || a.Count < 1 {
				return ErrBatchShape
			}
		}
	case ActKey:
		if (a.Type != KeyDown && a.Type != KeyUp) || a.Key == "" || len(a.Key) > 32 || len(a.Code) > 32 {
			return ErrBatchShape
		}
	case ActText, ActIME:
		if !utf8.ValidString(a.V) || (a.A == ActText && a.V == "") {
			return ErrBatchShape
		}
		n := utf8.RuneCountInString(a.V)
		if n > MaxText {
			return ErrBatchLimit
		}
		if a.Sel != nil && (a.A != ActIME || a.Sel[0] < 0 || a.Sel[0] > a.Sel[1] || a.Sel[1] > n) {
			return ErrBatchShape
		}
	default:
		return ErrBatchShape
	}
	return nil
}

func finite(f float64) bool { return !math.IsNaN(f) && !math.IsInf(f, 0) && math.Abs(f) <= maxCoord }
func coord(f float64) bool  { return finite(f) && f >= 0 }
