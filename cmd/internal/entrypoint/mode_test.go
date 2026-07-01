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
