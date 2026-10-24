package webcast

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
)

// Remote driving's routes (#466 specifies them, #467 serves them):
// POST <cast>/drive/hello takes a seat, /drive/control asks for or gives
// back control, /drive/input carries the driver's input. Answers are
// drive.Answer's. Without a controller the routes are not found.
//
// Budgets: hello per cast (a seat with no stream leaves after the grace
// period, so seats waiting for one are bounded by this rate times it, plus
// the burst; streams by the viewer cap). Control and input per seat, with
// four requests in flight each. The per-seat table drops a key the
// controller no longer knows when it is used, and, once it holds
// maxSeatLimits keys, every key idle past seatIdle (a seat's grace has
// long ended by then): bounded by live seats plus a minute of unknown keys.
const (
	helloRatePerSec   = 2
	helloBurst        = 8
	controlRatePerSec = 2
	controlBurst      = 5
	inputRatePerSec   = 30
	inputBurst        = 30
	seatInFlight      = broadcast.MaxPointInFlight
	maxHelloBody      = 512
	maxControlBody    = 128
	maxSeatLimits     = 256
	seatIdle          = time.Minute
)

type driveRoutes struct {
	ctrl   drive.Controller
	token  string // the drive link's
	hellos chan struct{}
	mu     sync.Mutex
	hello  *broadcast.RateLimit
	seats  map[string]*seatLimits // by key
	now    func() time.Time
	// logf is the operator's log (the cast's Logf; nil logs nothing), and
	// logged the messages already logged: in memory, at most maxLogged,
	// then forgotten and logged afresh.
	logf   func(format string, args ...any)
	warnf  func(format string, args ...any)
	logMu  sync.Mutex
	logged map[string]bool
}

const maxLogged = 64

func (d *driveRoutes) logOnce(msg string, actionable bool) {
	d.logMu.Lock()
	if d.logged[msg] {
		d.logMu.Unlock()
		return
	}
	if d.logged == nil || len(d.logged) >= maxLogged {
		d.logged = map[string]bool{}
	}
	d.logged[msg] = true
	logf := d.logf
	if actionable && d.warnf != nil {
		logf = d.warnf
	}
	d.logMu.Unlock()
	if logf != nil {
		logf("%s", msg)
	}
}

type seatLimits struct {
	used     time.Time
	inFlight chan struct{}
	control  *broadcast.RateLimit
	input    *broadcast.RateLimit
}

// WithDrive serves remote driving through ctrl under its own link's token
// (nil: no driving).
func (s *Server) WithDrive(ctrl drive.Controller, token string) *Server {
	if ctrl != nil && token != "" {
		s.drive = &driveRoutes{ctrl: ctrl, token: token, hellos: make(chan struct{}, seatInFlight),
			hello: broadcast.NewRateLimit(helloRatePerSec, helloBurst), seats: map[string]*seatLimits{}, now: time.Now}
	}
	return s
}

// WithDriveLog sends diagnostics to logf. The optional warning callback owns
// actionable failures; transient cancellation/busy responses stay informational.
func (s *Server) WithDriveLog(logf func(format string, args ...any), warnings ...func(string, ...any)) *Server {
	if s.drive != nil {
		s.drive.logMu.Lock()
		s.drive.logf = logf
		if len(warnings) > 0 {
			s.drive.warnf = warnings[0]
		}
		s.drive.logMu.Unlock()
	}
	return s
}

// DriveToken is the drive link's token ("" without remote driving).
func (s *Server) DriveToken() string {
	if s.drive == nil {
		return ""
	}
	return s.drive.token
}

// driveRoute serves one POST; false when rest isn't a drive route. Under
// the view-only and pointer links (driveLink false) the routes are not found.
func (s *Server) driveRoute(w http.ResponseWriter, r *http.Request, rest string, driveLink bool) bool {
	switch rest {
	case "drive/hello", "drive/control", "drive/input":
	default:
		return false
	}
	if !driveLink {
		http.NotFound(w, r)
		return true
	}
	d := s.drive
	if rest == "drive/hello" {
		body, release, ok := admitBody(w, r, d.hellos, d.allowHello, maxHelloBody)
		if !ok {
			return true
		}
		defer release()
		var in struct {
			Name string `json:"name"`
		}
		if !broadcast.DecodeStrict(body, &in) {
			d.answer(w, drive.ErrBatchShape)
			return true
		}
		seat, err := d.ctrl.Join(in.Name)
		if err != nil {
			d.answer(w, err)
			return true
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"viewer": seat.Viewer.ID, "key": seat.Key, "name": seat.Viewer.Name})
		return true
	}
	key := r.Header.Get(drive.SeatHeader)
	if key == "" {
		d.answer(w, drive.ErrNoSeat)
		return true
	}
	generation, err := strconv.ParseUint(r.Header.Get("X-Couch-Drive-Generation"), 10, 64)
	if err != nil {
		d.answer(w, drive.ErrDriveUnavailable)
		return true
	}
	admission, err := d.ctrl.AdmitDriveGeneration(generation)
	if err != nil {
		d.answer(w, err)
		return true
	}
	lim := d.limits(key)
	allow, max := lim.allowControl, int64(maxControlBody)
	if rest == "drive/input" {
		allow, max = lim.allowInput, drive.MaxBody
	}
	body, release, ok := admitBody(w, r, lim.inFlight, func(now time.Time) bool {
		d.mu.Lock()
		defer d.mu.Unlock()
		return allow(now)
	}, max)
	if !ok {
		return true
	}
	defer release()
	if rest == "drive/control" {
		var in struct {
			Op string `json:"op"`
		}
		switch {
		case !broadcast.DecodeStrict(body, &in):
			err = drive.ErrBatchShape
		case in.Op == "request":
			err = admission.Request(key)
		case in.Op == "release":
			err = admission.Release(key)
		default:
			err = drive.ErrBatchShape
		}
	} else {
		var b drive.Batch
		if b, err = drive.ParseBatch(body); err == nil {
			err = admission.Input(r.Context(), key, b)
		}
	}
	if errors.Is(err, drive.ErrNoSeat) {
		d.forget(key)
	}
	d.answer(w, err)
	return true
}

func (d *driveRoutes) allowHello(now time.Time) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.hello.Allow(now)
}

// limits is a seat key's budgets, made on first use.
func (d *driveRoutes) limits(key string) *seatLimits {
	d.mu.Lock()
	defer d.mu.Unlock()
	now := d.now()
	if l := d.seats[key]; l != nil {
		l.used = now
		return l
	}
	if len(d.seats) >= maxSeatLimits {
		for k, l := range d.seats {
			if now.Sub(l.used) > seatIdle {
				delete(d.seats, k)
			}
		}
		if len(d.seats) >= maxSeatLimits {
			// A minute's burst of keys: the newest goes unbudgeted
			// rather than growing the table; refused at once.
			return &seatLimits{used: now, inFlight: make(chan struct{}), control: broadcast.NewRateLimit(0, 0), input: broadcast.NewRateLimit(0, 0)}
		}
	}
	l := &seatLimits{used: now, inFlight: make(chan struct{}, seatInFlight),
		control: broadcast.NewRateLimit(controlRatePerSec, controlBurst),
		input:   broadcast.NewRateLimit(inputRatePerSec, inputBurst)}
	d.seats[key] = l
	return l
}

func (d *driveRoutes) forget(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.seats, key)
}

// allowControl and allowInput run under driveRoutes.mu.
func (l *seatLimits) allowControl(now time.Time) bool { return l.control.Allow(now) }
func (l *seatLimits) allowInput(now time.Time) bool   { return l.input.Allow(now) }

// answer writes err's response. A failure that isn't the viewer's (a 5xx:
// the tab couldn't take the input, the cast was busy, or an error nothing
// maps) is logged for the operator with its cause, once per distinct
// message (pair#505: every drive input answered 500 and nothing said why).
func (d *driveRoutes) answer(w http.ResponseWriter, err error) {
	status, text := drive.Answer(err)
	if status >= http.StatusInternalServerError {
		d.logOnce(fmt.Sprintf("remote driving: %d: %v", status, err), status != http.StatusServiceUnavailable)
	}
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}
	http.Error(w, text, status)
}
