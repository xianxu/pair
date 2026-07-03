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

	// The caller (the loop) resolves the marker's tag/agent + any rename before
	// calling, so planRestart takes the FINAL tag/agent. Default Alt+n: resume the
	// saved session onto the saved args.
	p := planRestart(RestartMarker{}, "work", "claude", saved)
	if p.DropConfig || p.ContinueSlug != "" {
		t.Fatalf("alt+n plan flags: %+v", p)
	}
	if p.Args.ForcedTag != "work" || p.Args.Agent != "claude" ||
		!reflect.DeepEqual(p.Args.AgentArgs, []string{"--flag", "--resume", "SID-1"}) {
		t.Fatalf("alt+n args = %+v", p.Args)
	}

	// Shift+Alt+N: fresh conversation → drop config, no resume token, no slug.
	pn := planRestart(RestartMarker{NewSession: true}, "work", "claude", saved)
	if !pn.DropConfig || pn.ContinueSlug != "" {
		t.Fatalf("new-session plan flags: %+v", pn)
	}
	if !reflect.DeepEqual(pn.Args.AgentArgs, []string{"--flag"}) {
		t.Fatalf("new-session args = %v (must not carry a resume token)", pn.Args.AgentArgs)
	}

	// #55 compaction re-entry: continue rides the new_session arm → drop config +
	// carry the slug for the draft re-seed (never a standalone arm — shell 1055).
	pc := planRestart(RestartMarker{NewSession: true, Continue: "demo-slug"}, "work", "claude", saved)
	if !pc.DropConfig || pc.ContinueSlug != "demo-slug" {
		t.Fatalf("continue re-entry = %+v", pc)
	}
}
