package broadcast

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"
)

// The remote pointer's input limits (#412).
const (
	MaxPointBody      = 4 << 10
	MaxPointsPerBatch = 64
	maxPointGrid      = 1000 // cols or rows; far past any real terminal
	PointRatePerSec   = 30
	// MaxPointInFlight bounds concurrent point requests per broadcast; more
	// get 429 before their body is read.
	MaxPointInFlight = 4
	pointReadBudget  = 2 * time.Second
)

var (
	ErrPointShape   = errors.New("broadcast: malformed point batch")
	ErrPointTooMany = errors.New("broadcast: too many points in one batch")
	ErrPointRange   = errors.New("broadcast: point outside the grid")
)

// PointBatch is one request from a pointer page: points in grid cells, on
// the cols×rows grid the page was showing. Down is true while a stroke
// continues (finger or button still down).
type PointBatch struct {
	Cols   int      `json:"cols"`
	Rows   int      `json:"rows"`
	Down   bool     `json:"down"`
	Points [][2]int `json:"points"`
}

// ParsePointBatch reads a request body strictly: one JSON object with only
// these fields, 1..MaxPointsPerBatch integer pairs, each inside the stated
// grid. Its errors are fixed values that never echo the input.
func ParsePointBatch(body []byte) (PointBatch, error) {
	var raw struct {
		Cols   *int             `json:"cols"`
		Rows   *int             `json:"rows"`
		Down   *bool            `json:"down"`
		Points *[][]json.Number `json:"points"`
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return PointBatch{}, ErrPointShape
	}
	if _, err := dec.Token(); err != io.EOF {
		return PointBatch{}, ErrPointShape
	}
	if raw.Cols == nil || raw.Rows == nil || raw.Down == nil || raw.Points == nil || len(*raw.Points) == 0 {
		return PointBatch{}, ErrPointShape
	}
	if len(*raw.Points) > MaxPointsPerBatch {
		return PointBatch{}, ErrPointTooMany
	}
	b := PointBatch{Cols: *raw.Cols, Rows: *raw.Rows, Down: *raw.Down}
	if b.Cols < 1 || b.Rows < 1 || b.Cols > maxPointGrid || b.Rows > maxPointGrid {
		return PointBatch{}, ErrPointRange
	}
	for _, p := range *raw.Points {
		if len(p) != 2 {
			return PointBatch{}, ErrPointShape
		}
		var xy [2]int
		for i, n := range p {
			v, err := n.Int64()
			if err != nil {
				return PointBatch{}, ErrPointShape
			}
			xy[i] = int(v)
			limit := b.Cols
			if i == 1 {
				limit = b.Rows
			}
			if v < 0 || v >= int64(limit) {
				return PointBatch{}, ErrPointRange
			}
		}
		b.Points = append(b.Points, xy)
	}
	return b, nil
}

// RateLimit is a token bucket. Pure: the caller supplies the clock and owns
// locking.
type RateLimit struct {
	perSec, burst float64
	tokens        float64
	last          time.Time
}

func NewRateLimit(perSec, burst int) *RateLimit {
	return &RateLimit{perSec: float64(perSec), burst: float64(burst), tokens: float64(burst)}
}

// Allow takes one token if one is available at now.
func (r *RateLimit) Allow(now time.Time) bool {
	if !r.last.IsZero() && now.After(r.last) {
		r.tokens = min(r.burst, r.tokens+now.Sub(r.last).Seconds()*r.perSec)
	}
	if r.last.IsZero() || now.After(r.last) {
		r.last = now
	}
	if r.tokens < 1 {
		return false
	}
	r.tokens--
	return true
}
