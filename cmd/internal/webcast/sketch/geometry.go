// Package sketch projects recorded pressure points without clipping original paths.
package sketch

import (
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"math"
)

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func validBox(b recording.Box) bool {
	for _, v := range b {
		if !finite(v) {
			return false
		}
	}
	return b[2] > 0 && b[3] > 0
}

// ProjectPoint maps between anchor boxes and preserves pressure. Points outside
// the original box stay outside; renderer clipping never modifies stored paths.
func ProjectPoint(p recording.SketchPoint, from, to recording.Box) (recording.SketchPoint, bool) {
	if !validBox(from) || !validBox(to) || !finite(p[0]) || !finite(p[1]) || !finite(p[2]) || p[2] < 0 || p[2] > 1 {
		return recording.SketchPoint{}, false
	}
	out := recording.SketchPoint{to[0] + (p[0]-from[0])/from[2]*to[2], to[1] + (p[1]-from[1])/from[3]*to[3], p[2]}
	if !finite(out[0]) || !finite(out[1]) {
		return recording.SketchPoint{}, false
	}
	return out, true
}

// PressureWidth is in page units; viewer scale applies afterward, independently
// of the anchor's aspect ratio. Invalid inputs return NaN.
func PressureWidth(base, pressure float64) float64 {
	if !finite(base) || base <= 0 || !finite(pressure) || pressure < 0 || pressure > 1 {
		return math.NaN()
	}
	return base * (.25 + .75*pressure)
}
