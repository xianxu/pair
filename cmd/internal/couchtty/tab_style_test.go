package couchtty

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestParseTabStyle(t *testing.T) {
	for _, tc := range []struct {
		input   string
		want    TabStyle
		invalid bool
	}{
		{"", TabStyleCollapse, false}, {"collapse", TabStyleCollapse, false}, {"multiline", TabStyleMultiline, false},
		{"Collapse", 0, true}, {"single", 0, true}, {" collapse", 0, true},
	} {
		got, err := ParseTabStyle(tc.input)
		if (err != nil) != tc.invalid || !tc.invalid && got != tc.want {
			t.Errorf("ParseTabStyle(%q) = %v, %v", tc.input, got, err)
		}
	}
}

func TestConsoleTabStyleConfiguration(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	if c.TabStyle() != DefaultTabStyle || DefaultTabStyle != TabStyleCollapse {
		t.Fatal("Console does not default to collapse")
	}
	c.SetTabStyle(TabStyleMultiline)
	if c.TabStyle() != TabStyleMultiline {
		t.Fatal("Console dropped explicit multiline")
	}
}
