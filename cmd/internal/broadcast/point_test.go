package broadcast

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParsePointBatch(t *testing.T) {
	ok := []struct {
		name string
		body string
		want PointBatch
	}{
		{"tap", `{"cols":80,"rows":24,"down":true,"points":[[3,4]]}`, PointBatch{Cols: 80, Rows: 24, Down: true, Points: [][2]int{{3, 4}}}},
		{"stroke", `{"cols":80,"rows":24,"down":false,"points":[[0,0],[79,22]]}`, PointBatch{Cols: 80, Rows: 24, Points: [][2]int{{0, 0}, {79, 22}}}},
	}
	for _, c := range ok {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParsePointBatch([]byte(c.body))
			if err != nil {
				t.Fatal(err)
			}
			if got.Cols != c.want.Cols || got.Rows != c.want.Rows || got.Down != c.want.Down || len(got.Points) != len(c.want.Points) {
				t.Fatalf("got %+v", got)
			}
			for i := range got.Points {
				if got.Points[i] != c.want.Points[i] {
					t.Fatalf("got %+v", got)
				}
			}
		})
	}
	many := `{"cols":80,"rows":24,"down":true,"points":[` + strings.TrimSuffix(strings.Repeat("[1,1],", MaxPointsPerBatch+1), ",") + `]}`
	bad := []struct {
		name string
		body string
		want error
	}{
		{"empty", ``, ErrPointShape},
		{"not json", `hello`, ErrPointShape},
		{"unknown field", `{"cols":80,"rows":24,"down":true,"points":[[1,1]],"keys":"rm -rf"}`, ErrPointShape},
		{"trailing data", `{"cols":80,"rows":24,"down":true,"points":[[1,1]]}{}`, ErrPointShape},
		{"missing points", `{"cols":80,"rows":24,"down":true}`, ErrPointShape},
		{"no points", `{"cols":80,"rows":24,"down":true,"points":[]}`, ErrPointShape},
		{"point not a pair", `{"cols":80,"rows":24,"down":true,"points":[[1,1,1]]}`, ErrPointShape},
		{"point not numbers", `{"cols":80,"rows":24,"down":true,"points":[["a",1]]}`, ErrPointShape},
		{"fractional", `{"cols":80,"rows":24,"down":true,"points":[[1.5,1]]}`, ErrPointShape},
		{"too many points", many, ErrPointTooMany},
		{"negative", `{"cols":80,"rows":24,"down":true,"points":[[-1,1]]}`, ErrPointRange},
		{"past cols", `{"cols":80,"rows":24,"down":true,"points":[[80,1]]}`, ErrPointRange},
		{"past rows", `{"cols":80,"rows":24,"down":true,"points":[[1,24]]}`, ErrPointRange},
		{"huge", `{"cols":80,"rows":24,"down":true,"points":[[1e300,1]]}`, ErrPointShape},
		{"zero grid", `{"cols":0,"rows":0,"down":true,"points":[[0,0]]}`, ErrPointRange},
		{"huge grid", `{"cols":100000,"rows":100000,"down":true,"points":[[0,0]]}`, ErrPointRange},
	}
	for _, c := range bad {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ParsePointBatch([]byte(c.body)); !errors.Is(err, c.want) {
				t.Fatalf("err %v, want %v", err, c.want)
			}
		})
	}
}

func TestRateLimit(t *testing.T) {
	r := NewRateLimit(30, 30)
	now := t0
	allowed := 0
	for range 100 {
		if r.Allow(now) {
			allowed++
		}
	}
	if allowed != 30 {
		t.Fatalf("burst allowed %d, want 30", allowed)
	}
	if r.Allow(now) {
		t.Fatal("allowed past an empty bucket")
	}
	now = now.Add(100 * time.Millisecond) // refills 3
	allowed = 0
	for range 10 {
		if r.Allow(now) {
			allowed++
		}
	}
	if allowed != 3 {
		t.Fatalf("after 100ms allowed %d, want 3", allowed)
	}
	if r.Allow(now.Add(-time.Second)) {
		t.Fatal("a clock going backwards refilled the bucket")
	}
}
