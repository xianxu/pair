package layoutcmd

import "testing"

func TestTerminalResizeTarget(t *testing.T) {
	tests := []struct {
		name         string
		terminalCols int
		screenCols   int
		want         int
		ok           bool
	}{
		{name: "at half targets two thirds", terminalCols: 75, screenCols: 150, want: 100, ok: true},
		{name: "just under expanded threshold targets two thirds", terminalCols: 89, screenCols: 150, want: 100, ok: true},
		{name: "at 60 percent reads expanded, targets half", terminalCols: 90, screenCols: 150, want: 75, ok: true},
		{name: "fully expanded targets half", terminalCols: 105, screenCols: 150, want: 75, ok: true},
		{name: "zero terminal refuses", terminalCols: 0, screenCols: 150},
		{name: "zero screen refuses", terminalCols: 75, screenCols: 0},
		{name: "terminal wider than screen refuses", terminalCols: 200, screenCols: 150},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := terminalResizeTarget(tt.terminalCols, tt.screenCols)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("terminalResizeTarget(%d, %d) = (%d, %v), want (%d, %v)",
					tt.terminalCols, tt.screenCols, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestTerminalResizeStep(t *testing.T) {
	tests := []struct {
		name    string
		current int
		target  int
		want    string
		done    bool
	}{
		{name: "below target grows leftward", current: 75, target: 100, want: "resize increase left"},
		{name: "above target shrinks leftward", current: 105, target: 75, want: "resize decrease left"},
		{name: "within tolerance is done", current: 99, target: 100, done: true},
		{name: "exactly at target is done", current: 100, target: 100, done: true},
		{name: "tolerance boundary is done", current: 98, target: 100, done: true},
		{name: "just past tolerance steps", current: 97, target: 100, want: "resize increase left"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, done := terminalResizeStep(tt.current, tt.target)
			got := ""
			for i, a := range action {
				if i > 0 {
					got += " "
				}
				got += a
			}
			if got != tt.want || done != tt.done {
				t.Fatalf("terminalResizeStep(%d, %d) = (%q, %v), want (%q, %v)",
					tt.current, tt.target, got, done, tt.want, tt.done)
			}
		})
	}
}
