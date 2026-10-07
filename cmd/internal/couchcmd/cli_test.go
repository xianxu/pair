package couchcmd

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
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
		{name: "reconcile", args: []string{"--reconcile", "pair:2"}, want: cliInvocation{kind: cliReconcile, ref: "pair:2"}},
		{name: "peek", args: []string{"--peek", "pair:1"}, want: cliInvocation{kind: cliPeek, ref: "pair:1"}},
		{name: "peek options", args: []string{"--peek", "pair:1", "--lines", "80", "--json"}, want: cliInvocation{kind: cliPeek, ref: "pair:1", args: []string{"--lines=80", "--json"}}},
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
		{"--peek"}, {"--peek", ""}, {"--peek", "--json"}, {"--peek", "pair:1", "--lines"},
		{"--peek", "pair:1", "--json", "--json"}, {"--peek", "pair:1", "--lines", "3", "--lines", "4"},
		{"--peek", "pair:1", "extra"}, {"--peek", "pair:1", "--layout3"},
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

func TestParseSlotOperationCLI(t *testing.T) {
	for _, tc := range []struct {
		args      []string
		op, ref   string
		confirmed bool
		json      bool
	}{
		{args: []string{"--resume", "pair:1"}, op: "resume", ref: "pair:1"},
		{args: []string{"--resume", "pair:0", "--json"}, op: "resume", ref: "pair:0", json: true},
		{args: []string{"--reboot", "pair:1", "--confirm"}, op: "reboot", ref: "pair:1", confirmed: true},
		{args: []string{"--reboot", "pa:1", "--confirm", "--json"}, op: "reboot", ref: "pa:1", confirmed: true, json: true},
		{args: []string{"--reboot", "pair:1", "--json", "--confirm"}, op: "reboot", ref: "pair:1", confirmed: true, json: true},
	} {
		got, err := ParseCLI(tc.args, couchcore.Operations())
		if err != nil || got.kind != cliMessage || got.messageOp != tc.op || got.ref != tc.ref || got.confirmed != tc.confirmed || got.jsonOutput != tc.json {
			t.Errorf("%q: %#v %v", tc.args, got, err)
		}
	}
}

// One strategy per malformation of the slot-operation argv.
func TestParseSlotOperationCLIRejectsMalformedArgv(t *testing.T) {
	for _, tc := range []struct {
		why  string
		args []string
	}{
		{"no target", []string{"--resume"}},
		{"a family is not one slot", []string{"--resume", "pair"}},
		{"a bare :N names no repository", []string{"--resume", ":1"}},
		{"a malformed slot number", []string{"--resume", "pair:01"}},
		{"a flag where the target goes", []string{"--resume", "--json"}},
		{"an extra positional", []string{"--resume", "pair:1", "extra"}},
		{"a repeated flag", []string{"--resume", "pair:1", "--json", "--json"}},
		{"confirmation on an operation that declares none", []string{"--resume", "pair:1", "--confirm"}},
		{"reboot without its declared confirmation", []string{"--reboot", "pair:1"}},
		{"a repeated confirmation", []string{"--reboot", "pair:1", "--confirm", "--confirm"}},
		{"a layout flag", []string{"--resume", "pair:1", "--layout2"}},
		{"a layout flag first", []string{"--layout2", "--resume", "pair:1"}},
	} {
		if got, err := ParseCLI(tc.args, couchcore.Operations()); err == nil || got.kind != cliInvalid {
			t.Errorf("%s: accepted %q: %#v", tc.why, tc.args, got)
		}
	}
}

// Every command text the report emits parses back to the operation and slot
// it names (pair#367 M2).
func TestSlotOperationCommandParses(t *testing.T) {
	for _, op := range []string{"resume", "reboot"} {
		got := parseCommandText(t, couchcore.SlotOperationCommand(op, "pair:2"))
		if got.kind != cliMessage || got.messageOp != op || got.ref != "pair:2" {
			t.Errorf("%s: %#v", op, got)
		}
	}
	message := couchcore.RestoreWorkspaceMessage("pair#000014", "pair:2", "ariadne")
	got := parseCommandText(t, couchcore.SendToCommand("pair:2", message))
	if got.messageOp != "send" || got.ref != "pair:2" || got.messageBody != message {
		t.Errorf("send-to: %#v", got)
	}
}

// TestRecoverPlanStepCommandsParse sweeps the steps of a real derivation: every
// emitted command parses, and a resume/reboot command names its row's slot.
func TestRecoverPlanStepCommandsParse(t *testing.T) {
	fake := couchcore.NewFakeFleetSDLC()
	fleet := fake.Fleet("/fleet")
	fleet.AddSlot("pair:1")
	fleet.SetBranch("pair:1", "000011-x")
	fleet.Claim("pair:1", "pair#000011")
	fleet.AddSlot("pair:2")
	fleet.Claim("pair:2", "pair#000012")
	fleet.AddSlot("pair:0")
	fleet.SetBranch("pair:0", "000010-x")
	fleet.Claim("pair:0", "pair#000010")
	var rows []couchcore.ActionableThreadSummary
	for _, c := range []struct {
		address string
		state   couchcore.ActionableThreadState
		reason  couchcore.ThreadReason
	}{{"pair:1", couchcore.ThreadParked, ""}, {"pair:2", couchcore.ThreadParked, ""}, {"pair:0", couchcore.ThreadUnusable, couchcore.ReasonSessionGone}} {
		path := fleet.SlotPath(c.address)
		scope, err := launcher.ResolveRepoScope(path)
		if err != nil {
			t.Fatal(err)
		}
		row := couchcore.ActionableThreadSummary{Address: couchcore.ThreadAddress{RepoScope: scope.Key, Tag: couchcore.ThreadTag("t" + c.address[len(c.address)-1:])},
			State: c.state, Reason: c.reason, StartingPath: path, WorkingPath: path, Target: couchcore.ThreadTarget{Kind: couchcore.ThreadTargetOrdinary}}
		if c.address != "pair:0" {
			n := int(c.address[len(c.address)-1] - '0')
			row.Target = couchcore.ThreadTarget{Kind: couchcore.ThreadTargetSlot, Slot: couchcore.SlotIdentity{Repo: "pair", Number: n, PrimaryRoot: "/fleet/pair", WorktreeRoot: path}}
		}
		rows = append(rows, row)
	}
	plan := couchcore.DeriveRecoverPlan(couchcore.RecoverPlanInput{
		Fleets: []couchcore.FleetObservation{{Root: "/fleet", State: couchcore.FleetObservationPresent, Inventory: fleet.Inventory()}},
		Couch:  couchcore.CouchObservation{State: couchcore.CouchObservationOK, Rows: rows},
	})
	seen := map[string]bool{}
	for _, row := range plan.Rows {
		for _, step := range row.Next.Steps {
			seen[step.Action] = true
			got := parseCommandText(t, step.Command)
			want := step.Action
			if step.Action == "ask-agent-restore" {
				want = "send"
			}
			if got.kind != cliMessage || got.messageOp != want || got.ref != row.Address {
				t.Errorf("%s step %s command %q parsed as %#v", row.Address, step.Action, step.Command, got)
			}
		}
	}
	for _, action := range []string{"resume", "reboot", "ask-agent-restore"} {
		if !seen[action] {
			t.Errorf("no row emitted a %s step: %+v", action, plan.Rows)
		}
	}
}

// parseCommandText shell-splits a `couch …` command line and parses its argv.
func parseCommandText(t *testing.T, command string) cliInvocation {
	t.Helper()
	argv, err := shellSplit(command)
	if err != nil || len(argv) == 0 || argv[0] != "couch" {
		t.Fatalf("command %q split as %q (%v)", command, argv, err)
	}
	got, err := ParseCLI(argv[1:], couchcore.Operations())
	if err != nil {
		t.Fatalf("command %q does not parse: %v", command, err)
	}
	return got
}

// shellSplit is a test helper splitting a POSIX shell word list: whitespace
// separates words, '…' quotes literally, "…" quotes with \" and \\ escapes,
// and a backslash outside quotes escapes the next character (so '\” works).
func shellSplit(s string) ([]string, error) {
	var words []string
	var word strings.Builder
	inWord := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'':
			end := strings.IndexByte(s[i+1:], '\'')
			if end < 0 {
				return nil, fmt.Errorf("unterminated single quote in %q", s)
			}
			word.WriteString(s[i+1 : i+1+end])
			i += end + 1
			inWord = true
		case c == '"':
			i++
			for ; i < len(s) && s[i] != '"'; i++ {
				if s[i] == '\\' && i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\\') {
					i++
				}
				word.WriteByte(s[i])
			}
			if i >= len(s) {
				return nil, fmt.Errorf("unterminated double quote in %q", s)
			}
			inWord = true
		case c == '\\' && i+1 < len(s):
			i++
			word.WriteByte(s[i])
			inWord = true
		case c == ' ' || c == '\t' || c == '\n':
			if inWord {
				words = append(words, word.String())
				word.Reset()
				inWord = false
			}
		default:
			word.WriteByte(c)
			inWord = true
		}
	}
	if inWord {
		words = append(words, word.String())
	}
	return words, nil
}

func TestShellSplit(t *testing.T) {
	for in, want := range map[string][]string{
		`couch --send-to pair:1 --message 'it'\''s here'`: {"couch", "--send-to", "pair:1", "--message", "it's here"},
		`couch --send-to pair:1 --message "a \"b\" c"`:    {"couch", "--send-to", "pair:1", "--message", `a "b" c`},
		`couch  --resume   pair:2`:                        {"couch", "--resume", "pair:2"},
	} {
		if got, err := shellSplit(in); err != nil || !slices.Equal(got, want) {
			t.Errorf("%s: %q %v", in, got, err)
		}
	}
}
