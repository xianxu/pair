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

// The flag is stripped before the shape checks, mirroring launcher.ParseArgs,
// which runs extractLayoutRequest first so the positional guard never sees a
// layout flag. That is why it composes with a path on either side.
func TestParseCLIAcceptsLayoutFlag(t *testing.T) {
	operations := couchcore.Operations()
	for _, tt := range []struct {
		name string
		args []string
		want cliInvocation
	}{
		{"bare defaults to layout2", nil,
			cliInvocation{kind: cliLaunch, path: ".", layout: couchcore.Layout2}},
		{"flag alone", []string{"--layout3"},
			cliInvocation{kind: cliLaunch, path: ".", layout: couchcore.Layout3}},
		{"flag before path", []string{"--layout3", "../pair"},
			cliInvocation{kind: cliLaunch, path: "../pair", layout: couchcore.Layout3}},
		{"flag after path", []string{"../pair", "--layout3"},
			cliInvocation{kind: cliLaunch, path: "../pair", layout: couchcore.Layout3}},
		{"explicit layout2", []string{"--layout2"},
			cliInvocation{kind: cliLaunch, path: ".", layout: couchcore.Layout2}},
		{"flag with a dash path", []string{"--layout3", "--", "-repo"},
			cliInvocation{kind: cliLaunch, path: "-repo", layout: couchcore.Layout3}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCLI(tt.args, operations)
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseCLI(%q) = %#v, %v; want %#v", tt.args, got, err, tt.want)
			}
		})
	}
}

func TestParseCLIRejectsContradictoryLayouts(t *testing.T) {
	if _, err := ParseCLI([]string{"--layout2", "--layout3"}, couchcore.Operations()); err == nil {
		t.Fatal("ParseCLI accepted two layouts at once")
	}
}

// Layout is a property of the couch SESSION being started, so it is meaningless
// on the read-only forms. Accepting it silently would teach the operator it
// does something.
func TestParseCLIRejectsLayoutOnNonLaunchForms(t *testing.T) {
	for _, args := range [][]string{
		{"--list", "--layout3"},
		{"--archived", "--layout3"},
		{"--show", "thread", "--layout3"},
		{"--help", "--layout3"},
		{"--internal", "publish-description", "--layout3"},
	} {
		if _, err := ParseCLI(args, couchcore.Operations()); err == nil {
			t.Fatalf("ParseCLI(%q) accepted a layout flag on a non-launch form", args)
		}
	}
}

func TestUsageMentionsTheLayoutFlag(t *testing.T) {
	var out strings.Builder
	usage(&out)
	if !strings.Contains(out.String(), "--layout3") {
		t.Fatalf("usage does not mention --layout3:\n%s", out.String())
	}
}

// End to end: `couch --layout3` must actually reach Couch.Layout. Without this,
// every ParseCLI test above could pass while the flag is parsed and then
// dropped on the floor between the CLI and the domain.
func TestLayoutFlagReachesTheCouch(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want couchcore.Layout
	}{
		{[]string{"--layout3"}, couchcore.Layout3},
		{nil, couchcore.Layout2},
	} {
		t.Run(string(tc.want), func(t *testing.T) {
			rt := newRT(t, "/repo")
			invocation, err := ParseCLI(tc.args, couchcore.Operations())
			if err != nil {
				t.Fatal(err)
			}
			master, slave, err := pty.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer master.Close()
			defer slave.Close()

			var seen couchcore.Layout
			finish := func(_ *couchtty.Console, c *couchcore.Couch, _ couchcore.StartResult, _ io.Writer) int {
				seen = c.Layout
				return 0
			}
			op, _ := Resolve("start")
			var stdout, stderr bytes.Buffer
			runTypedOperationWithConsole(op, map[string]string{}, map[string]string{"path": "/repo"},
				true, invocation.layout, slave, slave, slave, &stdout, &stderr, rt, finish)
			if seen != tc.want {
				t.Fatalf("ParseCLI(%q) reached the Couch as %q; want %q", tc.args, seen, tc.want)
			}
		})
	}
}
