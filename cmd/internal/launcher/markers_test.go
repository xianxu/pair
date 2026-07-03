package launcher

import (
	"reflect"
	"testing"
)

func TestParseRestartMarker(t *testing.T) {
	got := parseRestartMarker("tag=work\nagent=codex\nnew_session=1\n")
	want := RestartMarker{Tag: "work", Agent: "codex", NewSession: true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseRestartMarker = %+v, want %+v", got, want)
	}
	// continue + rename keys, and new_session!=1 → false.
	m := parseRestartMarker("tag=t\nnew_session=0\nrename_to=newt\ncontinue=slug-1\n")
	if m.NewSession || m.RenameTo != "newt" || m.Continue != "slug-1" {
		t.Fatalf("parseRestartMarker mixed = %+v", m)
	}
}

func TestPlanRestart(t *testing.T) {
	saved := savedConfig{Agent: "claude", Args: []string{"--flag"}, SessionID: "SID-1"}

	// Default Alt+n: resume the saved session onto the saved args.
	p := planRestart(RestartMarker{Tag: "work", Agent: "claude"}, "cur", "curagent", saved)
	if p.ShellFallback || p.DropConfig {
		t.Fatalf("alt+n plan flags: %+v", p)
	}
	if p.Args.ForcedTag != "work" || p.Args.Agent != "claude" ||
		!reflect.DeepEqual(p.Args.AgentArgs, []string{"--flag", "--resume", "SID-1"}) {
		t.Fatalf("alt+n args = %+v", p.Args)
	}

	// Shift+Alt+N: fresh conversation → drop config, no resume token.
	pn := planRestart(RestartMarker{Tag: "work", Agent: "claude", NewSession: true}, "cur", "curagent", saved)
	if !pn.DropConfig || pn.ShellFallback {
		t.Fatalf("new-session plan flags: %+v", pn)
	}
	if !reflect.DeepEqual(pn.Args.AgentArgs, []string{"--flag"}) {
		t.Fatalf("new-session args = %v (must not carry a resume token)", pn.Args.AgentArgs)
	}

	// Marker tag/agent default to the current run's when unset.
	pd := planRestart(RestartMarker{}, "curtag", "curagent", savedConfig{})
	if pd.Args.ForcedTag != "curtag" || pd.Args.Agent != "curagent" {
		t.Fatalf("defaulting = %+v", pd.Args)
	}

	// rename_to / continue → shell fallback (M5-coupled re-entries).
	if !planRestart(RestartMarker{RenameTo: "x"}, "t", "a", saved).ShellFallback {
		t.Fatal("rename_to should fall back")
	}
	if !planRestart(RestartMarker{Continue: "s"}, "t", "a", saved).ShellFallback {
		t.Fatal("continue should fall back")
	}
}
