package entrypoint

import "testing"

func TestBusyboxSubcommand(t *testing.T) {
	// valid mirrors dispatcher.DispatchNames() in the nested world: flat leaves
	// plus the group tokens. Only flat names are reachable via a busybox base
	// name (the surviving case is the external pair-slug Stop-hook symlink).
	valid := []string{"context", "slug", "wrap", "scribe", "session-watch", "title",
		"continuation", "review", "scrollback", "changelog", "clip"}
	cases := []struct {
		base    string
		wantSub string
		wantOK  bool
	}{
		{"pair-slug", "slug", true},        // the one surviving external symlink
		{"pair-title", "title", true},      // any flat pair-<x> resolves
		{"pair-context", "context", true},
		{"pair", "", false},                // launcher, its own entrypoint
		{"pair-go", "", false},             // launch handoff
		{"pair-dev", "", false},            // dev wrapper shell shim
		{"pair-scribe", "", false},         // folds to `pair scribe`, not a busybox alias
		{"pair-help", "", false},           // shell shim
		{"pair-notify", "", false},         // shell shim
		{"pair-review-open", "", false},    // nested family — never a busybox name
		{"pair-frob", "", false},           // pair-<x> but x not implemented
		{"randomtool", "", false},          // arbitrary PATH tool never resolves
		{"slug", "", false},                // bare token without pair- prefix never resolves
		{"title", "", false},               // ditto
	}
	for _, c := range cases {
		sub, ok := busyboxSubcommand(c.base, valid)
		if sub != c.wantSub || ok != c.wantOK {
			t.Errorf("busyboxSubcommand(%q) = (%q, %v), want (%q, %v)", c.base, sub, ok, c.wantSub, c.wantOK)
		}
	}
}
