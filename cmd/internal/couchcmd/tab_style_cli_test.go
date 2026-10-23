package couchcmd

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

func TestTabStyleCLI(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		style couchtty.TabStyle
		path  string
	}{
		{nil, couchtty.TabStyleCollapse, "."},
		{[]string{"--tab-style", "multiline"}, couchtty.TabStyleMultiline, "."},
		{[]string{"--tab-style=collapse", "../pair", "--layout2"}, couchtty.TabStyleCollapse, "../pair"},
		{[]string{"../pair", "--tab-style", "multiline", "--layout3"}, couchtty.TabStyleMultiline, "../pair"},
		{[]string{"--tab-style=multiline", "--", "--tab-style=collapse"}, couchtty.TabStyleMultiline, "--tab-style=collapse"},
	} {
		inv, err := ParseCLI(tc.args, couchcore.Operations())
		if err != nil || inv.kind != cliLaunch || inv.tabStyle != tc.style || inv.path != tc.path {
			t.Errorf("ParseCLI(%q) = %#v, %v", tc.args, inv, err)
		}
	}
	for _, args := range [][]string{
		{"--tab-style"}, {"--tab-style=bogus"}, {"--tab-style", ""},
		{"--tab-style=collapse", "--tab-style=multiline"},
		{"--list", "--tab-style=collapse"}, {"--tab-style=collapse", "--show", "pair:1"},
		{"--tab-style=collapse", "--send-to", "pair:1", "--message", "body"},
	} {
		if _, err := ParseCLI(args, couchcore.Operations()); err == nil {
			t.Errorf("accepted %q", args)
		}
	}
	inv, err := ParseCLI([]string{"--send-to", "pair:1", "--message", "--tab-style=multiline"}, couchcore.Operations())
	if err != nil || inv.messageBody != "--tab-style=multiline" {
		t.Fatalf("message body changed: %#v, %v", inv, err)
	}
}

func TestExtractTabStyleFlag(t *testing.T) {
	rest, style, given, err := extractTabStyleFlag([]string{"repo", "--tab-style=multiline", "--tab-style", "multiline", "--", "--tab-style=collapse"})
	if err != nil || !given || style != couchtty.TabStyleMultiline || !reflect.DeepEqual(rest, []string{"repo", "--", "--tab-style=collapse"}) {
		t.Fatalf("extract = %q, %v, %v, %v", rest, style, given, err)
	}
}

func TestTabStyleReachesProductionConsole(t *testing.T) {
	for _, args := range [][]string{nil, {"--tab-style=multiline"}} {
		t.Run("style", func(t *testing.T) {
			inv, err := ParseCLI(args, couchcore.Operations())
			if err != nil {
				t.Fatal(err)
			}
			rt := newRT(t, "/repo")
			master, slave, err := pty.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer master.Close()
			defer slave.Close()
			if err := pty.Setsize(slave, &pty.Winsize{Rows: 24, Cols: 80}); err != nil {
				t.Fatal(err)
			}
			seen := false
			finish := func(c *couchtty.Console, _ *couchcore.Couch, _ couchcore.StartResult, _ io.Writer) int {
				seen = true
				if c.TabStyle() != inv.tabStyle {
					t.Errorf("Console style=%v, want %v", c.TabStyle(), inv.tabStyle)
				}
				return 0
			}
			op, _ := Resolve("start")
			var stdout, stderr bytes.Buffer
			runTypedOperationWithConsoleStyle(op, map[string]string{}, map[string]string{"path": "/repo"}, true, inv.layout, inv.tabStyle, slave, slave, slave, &stdout, &stderr, rt, finish)
			if !seen {
				t.Fatalf("Console not reached: %s", stderr.String())
			}
		})
	}
}

func TestTabStyleHelp(t *testing.T) {
	var out strings.Builder
	usage(&out)
	if !strings.Contains(out.String(), "--tab-style collapse") || !strings.Contains(out.String(), "--tab-style multiline") || !strings.Contains(out.String(), "default") {
		t.Fatalf("help omits tab styles and default: %s", out.String())
	}
}
