package webcast

import (
	"encoding/json"
	"errors"
	"net/http"
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
// four requests in flight each; the per-seat table is bounded (cleared
// past maxSeatLimits) and drops a seat the controller no longer knows.
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
)

type driveRoutes struct {
	ctrl   drive.Controller
	hellos chan struct{}
	mu     sync.Mutex
	hello  *broadcast.RateLimit
	seats  map[string]*seatLimits // by key
}

type seatLimits struct {
	inFlight chan struct{}
	control  *broadcast.RateLimit
	input    *broadcast.RateLimit
}

// WithDrive serves remote driving through ctrl (nil: no driving).
func (s *Server) WithDrive(ctrl drive.Controller) *Server {
	if ctrl != nil {
		s.drive = &driveRoutes{ctrl: ctrl, hellos: make(chan struct{}, seatInFlight),
			hello: broadcast.NewRateLimit(helloRatePerSec, helloBurst), seats: map[string]*seatLimits{}}
	}
	return s
}

// driveRoute serves one POST; false when rest isn't a drive route.
func (s *Server) driveRoute(w http.ResponseWriter, r *http.Request, rest string) bool {
	switch rest {
	case "drive/hello", "drive/control", "drive/input":
	default:
		return false
	}
	if s.drive == nil {
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
			answerDrive(w, drive.ErrBatchShape)
			return true
		}
		seat, err := d.ctrl.Join(in.Name)
		if err != nil {
			answerDrive(w, err)
			return true
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"viewer": seat.Viewer.ID, "key": seat.Key, "name": seat.Viewer.Name})
		return true
	}
	key := r.Header.Get(drive.SeatHeader)
	if key == "" {
		answerDrive(w, drive.ErrNoSeat)
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
	var err error
	if rest == "drive/control" {
		var in struct {
			Op string `json:"op"`
		}
		switch {
		case !broadcast.DecodeStrict(body, &in):
			err = drive.ErrBatchShape
		case in.Op == "request":
			err = d.ctrl.Request(key)
		case in.Op == "release":
			err = d.ctrl.Release(key)
		default:
			err = drive.ErrBatchShape
		}
	} else {
		var b drive.Batch
		if b, err = drive.ParseBatch(body); err == nil {
			err = d.ctrl.Input(r.Context(), key, b)
		}
	}
	if errors.Is(err, drive.ErrNoSeat) {
		d.forget(key)
	}
	answerDrive(w, err)
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
	if l := d.seats[key]; l != nil {
		return l
	}
	if len(d.seats) >= maxSeatLimits {
		d.seats = map[string]*seatLimits{} // a burst of unknown keys: start over
	}
	l := &seatLimits{inFlight: make(chan struct{}, seatInFlight),
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

func answerDrive(w http.ResponseWriter, err error) {
	status, text := drive.Answer(err)
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}
	http.Error(w, text, status)
}
