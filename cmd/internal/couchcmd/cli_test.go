package couchcmd

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// Since #198 every launch form carries a layout: absent means the default,
// Layout3, not "unset". The read-only forms carry none and refuse the flag.
func TestParseCLI(t *testing.T) {
	operations := couchcore.Operations()
	tests := []struct {
		name string
		args []string
		want cliInvocation
	}{
		{name: "bare", want: cliInvocation{kind: cliLaunch, path: ".", layout: couchcore.Layout3}},
		{name: "path", args: []string{"../pair"}, want: cliInvocation{kind: cliLaunch, path: "../pair", layout: couchcore.Layout3}},
		{name: "dash path", args: []string{"--", "-repo"}, want: cliInvocation{kind: cliLaunch, path: "-repo", layout: couchcore.Layout3}},
		{name: "list", args: []string{"--list"}, want: cliInvocation{kind: cliList}},
		{name: "recover plan", args: []string{"--recover-plan-from-sdlc"}, want: cliInvocation{kind: cliRecoverPlan}},
		{name: "show", args: []string{"--show", "thread"}, want: cliInvocation{kind: cliShow, ref: "thread"}},
		{name: "help long", args: []string{"--help"}, want: cliInvocation{kind: cliHelp}},
		{name: "help short", args: []string{"-h"}, want: cliInvocation{kind: cliHelp}},
		{name: "internal", args: []string{"--internal", "publish-description", "working"}, want: cliInvocation{kind: cliInternal, operation: "publish-description", args: []string{"working"}}},
		{name: "internal clear", args: []string{"--internal", "publish-description", ""}, want: cliInvocation{kind: cliInternal, operation: "publish-description", args: []string{""}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCLI(tt.args, operations)
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseCLI(%q) = %#v, %v; want %#v", tt.args, got, err, tt.want)
			}
		})
	}
}

func TestParseCLIRejectsMalformedOrUnpublishedForms(t *testing.T) {
	operations := couchcore.Operations()
	for _, args := range [][]string{
		{""}, {"a", "b"}, {"--"}, {"--", ""}, {"--", "a", "b"},
		{"--list", "x"}, {"--show"}, {"--show", ""}, {"--show", "x", "y"},
		{"--show", "--list"}, {"--show", "--help"}, {"--show", "--unknown"},
		{"--help", "x"}, {"-h", "x"}, {"--unknown"}, {"--agent=claude"},
		{"--internal"}, {"--internal=publish-description"}, {"--internal", ""},
		{"--internal", "list"}, {"--internal", "start"}, {"--internal", "missing"},
		{"--internal", "publish-description", "--"},
	} {
		got, err := ParseCLI(args, operations)
		if err == nil || got.kind != cliInvalid {
			t.Errorf("ParseCLI(%q) = %#v, %v; want invalid error", args, got, err)
		}
	}
}

func FuzzParseCLIIsClosed(f *testing.F) {
	for _, seed := range []string{"", "start", "--list", "--show", "--internal", "--help", "-x"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, token string) {
		got, err := ParseCLI([]string{token}, couchcore.Operations())
		if err != nil {
			if got.kind != cliInvalid {
				t.Fatalf("error returned partial invocation %#v", got)
			}
			return
		}
		if got.kind <= cliInvalid || got.kind > cliHelp {
			t.Fatalf("successful parse escaped closed kinds: %#v", got)
		}
	})
}

func TestParseMessageCLI(t *testing.T) {
	for _, tc := range []struct {
		args          []string
		kind          cliKind
		op, ref, body string
		agent         string
		json          bool
	}{
		{args: []string{"--actors"}, kind: cliMessage, op: "actors"},
		{args: []string{"--actors", "--json"}, kind: cliMessage, op: "actors", json: true},
		{args: []string{"--message-status", "id", "--json"}, kind: cliMessage, op: "status", ref: "id", json: true},
		{args: []string{"--send-to", "pair:1", "--message", "hello"}, kind: cliMessage, op: "send", ref: "pair:1", body: "hello"},
		{args: []string{"--send-to", "pair", "--message", "--layout2"}, kind: cliMessage, op: "send", ref: "pair", body: "--layout2"},
		{args: []string{"--send-to", "pair", "--message", "--actors\n--layout3"}, kind: cliMessage, op: "send", ref: "pair", body: "--actors\n--layout3"},
		{args: []string{"--send-to", "pair", "--agent", "codex", "--message", "hello"}, kind: cliMessage, op: "send", ref: "pair", agent: "codex", body: "hello"},
		{args: []string{"--send-to", "pair", "--agent", "codex", "--message", "--agent"}, kind: cliMessage, op: "send", ref: "pair", agent: "codex", body: "--agent"},
		{args: []string{"--skill"}, kind: cliSkill},
	} {
		got, err := ParseCLI(tc.args, couchcore.Operations())
		if err != nil || got.kind != tc.kind || got.messageOp != tc.op || got.ref != tc.ref || got.messageBody != tc.body || got.messageAgent != tc.agent || got.jsonOutput != tc.json {
			t.Errorf("%q: %#v %v", tc.args, got, err)
		}
	}
}

func TestParseMessageCLIRejectsMixedShapes(t *testing.T) {
	for _, args := range [][]string{
		{"--actors", "--json", "--json"}, {"--actors", "--send-to", "pair"}, {"--actors", "--layout2"},
		{"--layout2", "--actors"}, {"--message-status"}, {"--message-status", "id", "--send-to", "pair"},
		{"--send-to", "pair"}, {"--send-to", "pair", "--message", ""}, {"--send-to", "", "--message", "hello"},
		{"--send-to", "pair", "--message", "hello", "--json"}, {"--send-to", "pair", "--message", "hello", "--layout2"},
		{"--send-to", "pair", "--agent", "", "--message", "hello"}, {"--send-to", "pair", "--agent", "--message", "hello"},
		{"--send-to", "pair", "--message", "hello", "--agent", "codex"}, {"--send-to", "pair", "--agent", "-x", "--message", "hello"},
		{"--skill", "--json"}, {"--skill", "--layout2"}, {"--available", "on"}, {"--reply-to", "id", "--message", "hello"},
	} {
		if got, err := ParseCLI(args, couchcore.Operations()); err == nil || got.kind != cliInvalid {
			t.Errorf("accepted %q: %#v %v", args, got, err)
		}
	}
}

func TestRecoverPlanCLIRejectsArguments(t *testing.T) {
	for _, args := range [][]string{{"--recover-plan-from-sdlc", "x"}, {"--recover-plan-from-sdlc", "--layout2"}, {"--layout3", "--recover-plan-from-sdlc"}} {
		var out, diag bytes.Buffer
		if code := RunWithRuntime(args, strings.NewReader(""), &out, &diag, newRT(t)); code != 2 || out.Len() != 0 || diag.Len() == 0 {
			t.Errorf("%q: code=%d stdout=%q stderr=%q, want a usage refusal", args, code, &out, &diag)
		}
	}
}
