package couchcmd

import (
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestParseCLI(t *testing.T) {
	operations := couchcore.Operations()
	tests := []struct {
		name string
		args []string
		want cliInvocation
	}{
		{name: "bare", want: cliInvocation{kind: cliLaunch, path: "."}},
		{name: "path", args: []string{"../pair"}, want: cliInvocation{kind: cliLaunch, path: "../pair"}},
		{name: "dash path", args: []string{"--", "-repo"}, want: cliInvocation{kind: cliLaunch, path: "-repo"}},
		{name: "list", args: []string{"--list"}, want: cliInvocation{kind: cliList}},
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
