package layoutcmd

import (
	"strings"
	"testing"
)

func TestTerminalToggleBurst(t *testing.T) {
	tests := []struct {
		name         string
		terminalCols int
		screenCols   int
		want         string
		ok           bool
	}{
		{name: "at half expands", terminalCols: 75, screenCols: 150,
			want: "resize increase left,resize increase left,resize increase left", ok: true},
		{name: "just under expanded threshold expands", terminalCols: 89, screenCols: 150,
			want: "resize increase left,resize increase left,resize increase left", ok: true},
		{name: "at 60 percent collapses", terminalCols: 90, screenCols: 150,
			want: "resize decrease left,resize decrease left,resize decrease left", ok: true},
		{name: "fully expanded collapses", terminalCols: 105, screenCols: 150,
			want: "resize decrease left,resize decrease left,resize decrease left", ok: true},
		{name: "zero terminal refuses", terminalCols: 0, screenCols: 150},
		{name: "zero screen refuses", terminalCols: 75, screenCols: 0},
		{name: "terminal wider than screen refuses", terminalCols: 200, screenCols: 150},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			burst, ok := terminalToggleBurst(tt.terminalCols, tt.screenCols)
			var got []string
			for _, action := range burst {
				got = append(got, strings.Join(action, " "))
			}
			if strings.Join(got, ",") != tt.want || ok != tt.ok {
				t.Fatalf("terminalToggleBurst(%d, %d) = (%q, %v), want (%q, %v)",
					tt.terminalCols, tt.screenCols, strings.Join(got, ","), ok, tt.want, tt.ok)
			}
		})
	}
}
