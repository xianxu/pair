package entrypoint

import "testing"

func TestClassifyInvocation(t *testing.T) {
	names := []string{"slug", "changelog", "continuation", "session-watch", "context", "scrollback-render"}
	cases := []struct {
		name string
		exe  string
		args []string
		want EntrypointMode
	}{
		{"pair peels off a dispatch subcommand", "/x/bin/pair", []string{"slug"}, ModeDispatch},
		{"pair peels off with subcommand args", "/x/bin/pair", []string{"changelog", "--log", "f"}, ModeDispatch},
		{"pair context peels off", "/x/bin/pair", []string{"context", "T", "claude"}, ModeDispatch},
		{"agent name still launches", "/x/bin/pair", []string{"claude"}, ModePublicPair},
		{"launcher verb resume still launches", "/x/bin/pair", []string{"resume"}, ModePublicPair},
		{"launcher verb rename still launches", "/x/bin/pair", []string{"rename", "foo"}, ModePublicPair},
		{"bare pair launches", "/x/bin/pair", nil, ModePublicPair},
		{"pair-go dispatches subcommands", "/x/bin/pair-go", []string{"slug"}, ModeDispatch},
		{"pair-go launch is a handoff", "/x/bin/pair-go", []string{"launch"}, ModePairGoLaunch},
		{"pair-go bare is dispatch (help)", "/x/bin/pair-go", nil, ModeDispatch},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ClassifyInvocation(c.exe, c.args, names); got != c.want {
				t.Errorf("ClassifyInvocation(%q, %v) = %v, want %v", c.exe, c.args, got, c.want)
			}
		})
	}
}

func TestResolveInvocation(t *testing.T) {
	names := []string{"slug", "changelog", "context", "review", "scrollback", "clip"}
	cases := []struct {
		name     string
		exe      string
		args     []string
		wantMode EntrypointMode
		wantArgs []string
	}{
		{"pair dispatch passes args through", "/x/bin/pair", []string{"slug"}, ModeDispatch, []string{"slug"}},
		{"pair nested group passes through", "/x/bin/pair", []string{"review", "open"}, ModeDispatch, []string{"review", "open"}},
		{"pair launch stays public", "/x/bin/pair", []string{"claude"}, ModePublicPair, []string{"claude"}},
		{"busybox pair-slug prepends subcommand", "/x/bin/pair-slug", nil, ModeDispatch, []string{"slug"}},
		{"busybox pair-slug keeps its args", "/x/bin/pair-slug", []string{"--foo"}, ModeDispatch, []string{"slug", "--foo"}},
		{"pair-go launch is a handoff", "/x/bin/pair-go", []string{"launch"}, ModePairGoLaunch, []string{"launch"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotMode, gotArgs := ResolveInvocation(c.exe, c.args, names)
			if gotMode != c.wantMode {
				t.Errorf("mode = %v, want %v", gotMode, c.wantMode)
			}
			if len(gotArgs) != len(c.wantArgs) {
				t.Fatalf("args = %v, want %v", gotArgs, c.wantArgs)
			}
			for i := range gotArgs {
				if gotArgs[i] != c.wantArgs[i] {
					t.Fatalf("args = %v, want %v", gotArgs, c.wantArgs)
				}
			}
		})
	}
}
