package launcher

import "testing"

// reapRow is the fully-evidenced candidate: every predicate holds, so it kills.
// Each case below moves exactly one field to its boundary.
func reapRow() NvimReapRow {
	return NvimReapRow{
		Kind:            "draft",
		Owner:           "work",
		PID:             4242,
		PPID:            1,
		Identity:        "1700000000.000001",
		Name:            "/opt/homebrew/bin/nvim",
		Args:            []string{"nvim", "--embed", "-u", "/pair/nvim/init.lua", "/data/repos/scope/draft-work.md"},
		ArgsKnown:       true,
		OwnerLive:       false,
		ExpectedArgPath: "/data/repos/scope/draft-work.md",
	}
}

// #583: every field of a row is untrusted input, so the table drives one field
// at a time to its boundary — absent, wrong-valued, right-valued — and pins the
// verdict and the reason. The invariant is that no row missing any one
// predicate is ever a kill: a mis-read or a recycled pid must err toward
// leaving the process alone.
func TestClassifyNvimReapKillsOnlyOnCompleteIdentity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*NvimReapRow)
		kill   bool
		reason string
	}{
		{name: "owner dead", kill: true, reason: "owner dead"},
		{
			name:   "owner live but reparented to launchd",
			mutate: func(r *NvimReapRow) { r.OwnerLive = true; r.PPID = 1 },
			kill:   true,
			reason: "reparented to launchd",
		},
		{
			name:   "owner dead and not reparented",
			mutate: func(r *NvimReapRow) { r.PPID = 4243 },
			kill:   true,
			reason: "owner dead",
		},
		{
			name:   "owner live with a live parent",
			mutate: func(r *NvimReapRow) { r.OwnerLive = true; r.PPID = 4243 },
			reason: "owner live",
		},
		{
			name:   "owner live with an unknown parent",
			mutate: func(r *NvimReapRow) { r.OwnerLive = true; r.PPID = 0 },
			reason: "owner live",
		},
		{name: "no pid", mutate: func(r *NvimReapRow) { r.PID = 0 }, reason: "no pid"},
		{name: "negative pid", mutate: func(r *NvimReapRow) { r.PID = -1 }, reason: "no pid"},
		{name: "no owner", mutate: func(r *NvimReapRow) { r.Owner = "" }, reason: "no owner"},
		{
			name:   "no identity",
			mutate: func(r *NvimReapRow) { r.Identity = "" },
			reason: "identity unknown",
		},
		{name: "not nvim", mutate: func(r *NvimReapRow) { r.Name = "/bin/zsh" }, reason: "not nvim"},
		{name: "no name", mutate: func(r *NvimReapRow) { r.Name = "" }, reason: "not nvim"},
		{
			name:   "nvim in the name but not the whole name",
			mutate: func(r *NvimReapRow) { r.Name = "/bin/nvim-wrapper" },
			reason: "not nvim",
		},
		{
			name:   "argv unknown",
			mutate: func(r *NvimReapRow) { r.ArgsKnown = false },
			reason: "argv unknown",
		},
		{
			name:   "argv unreadable and empty",
			mutate: func(r *NvimReapRow) { r.ArgsKnown = false; r.Args = nil },
			reason: "argv unknown",
		},
		{
			name:   "no embed field",
			mutate: func(r *NvimReapRow) { r.Args = []string{"nvim", "/data/repos/scope/draft-work.md"} },
			reason: "no --embed field",
		},
		{
			name:   "empty argv",
			mutate: func(r *NvimReapRow) { r.Args = nil },
			reason: "no --embed field",
		},
		{
			name:   "embed only inside a longer field",
			mutate: func(r *NvimReapRow) { r.Args = []string{"nvim", "--embed-x", "/data/repos/scope/draft-work.md"} },
			reason: "no --embed field",
		},
		{
			name:   "embed as a substring of a path",
			mutate: func(r *NvimReapRow) { r.Args = []string{"nvim", "/data/--embed/draft-work.md"} },
			reason: "no --embed field",
		},
		{
			name:   "argv serves another draft",
			mutate: func(r *NvimReapRow) { r.Args = []string{"nvim", "--embed", "/data/repos/scope/draft-other.md"} },
			reason: "argv serves another path",
		},
		{
			name:   "expected path only inside a longer field",
			mutate: func(r *NvimReapRow) { r.Args = []string{"nvim", "--embed", "/data/repos/scope/draft-work.md.bak"} },
			reason: "argv serves another path",
		},
		{
			name:   "review kind expects no scope path",
			mutate: func(r *NvimReapRow) { r.Kind = "review"; r.ExpectedArgPath = ""; r.Args = []string{"nvim", "--embed"} },
			kill:   true,
			reason: "owner dead",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			row := reapRow()
			if test.mutate != nil {
				test.mutate(&row)
			}
			kill, reason := ClassifyNvimReap(row)
			if kill != test.kill || reason != test.reason {
				t.Fatalf("ClassifyNvimReap(%+v) = %v, %q, want %v, %q", row, kill, reason, test.kill, test.reason)
			}
			if reason == "" {
				t.Fatalf("ClassifyNvimReap(%+v) returned no reason", row)
			}
		})
	}
}

// The invariant, stated as its own assertion: dropping any single predicate
// from the fully-evidenced row must never leave it a kill.
func TestClassifyNvimReapNeverKillsOnMissingEvidence(t *testing.T) {
	t.Parallel()

	drop := map[string]func(*NvimReapRow){
		"pid":         func(r *NvimReapRow) { r.PID = 0 },
		"owner":       func(r *NvimReapRow) { r.Owner = "" },
		"identity":    func(r *NvimReapRow) { r.Identity = "" },
		"name":        func(r *NvimReapRow) { r.Name = "" },
		"argv known":  func(r *NvimReapRow) { r.ArgsKnown = false },
		"argv":        func(r *NvimReapRow) { r.Args = nil },
		"embed field": func(r *NvimReapRow) { r.Args = []string{"nvim"} },
		"expected":    func(r *NvimReapRow) { r.Args = []string{"nvim", "--embed"} },
		"live owner":  func(r *NvimReapRow) { r.OwnerLive = true; r.PPID = 99 },
	}
	for name, mutate := range drop {
		row := reapRow()
		mutate(&row)
		if kill, reason := ClassifyNvimReap(row); kill {
			t.Fatalf("%s: killed on missing evidence (%q)", name, reason)
		}
	}
}
