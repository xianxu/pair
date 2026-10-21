package sketch

import (
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"math"
	"os"
	"testing"
)

func TestSharedGeometry(t *testing.T) {
	raw, err := os.ReadFile("../../../../tests/webcast/sketch-geometry.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Points []struct {
			Name     string
			Point    []float64
			From, To recording.Box
			Want     *recording.SketchPoint
		}
		Widths []struct {
			Base, Pressure float64
			Want           *float64
		}
	}
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, c := range fixture.Points {
		t.Run(c.Name, func(t *testing.T) {
			p := recording.SketchPoint{c.Point[0], c.Point[1], 1}
			if len(c.Point) == 3 {
				p[2] = c.Point[2]
			}
			got, ok := ProjectPoint(p, c.From, c.To)
			if c.Want == nil {
				if ok {
					t.Fatalf("unexpected point %v", got)
				}
				return
			}
			if !ok || got != *c.Want {
				t.Fatalf("got %v %v, want %v", got, ok, *c.Want)
			}
		})
	}
	for _, c := range fixture.Widths {
		got := PressureWidth(c.Base, c.Pressure)
		if c.Want == nil {
			if !math.IsNaN(got) {
				t.Errorf("invalid width %v", got)
			}
		} else if got != *c.Want {
			t.Errorf("width %v want %v", got, *c.Want)
		}
	}
}
func TestNonfiniteGeometry(t *testing.T) {
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, ok := ProjectPoint(recording.SketchPoint{v, 1, 1}, recording.Box{0, 0, 10, 10}, recording.Box{0, 0, 10, 10}); ok {
			t.Fatal("nonfinite point accepted")
		}
		if !math.IsNaN(PressureWidth(4, v)) {
			t.Fatal("nonfinite pressure accepted")
		}
	}
}
