package launcher

import (
	"reflect"
	"testing"
)

func TestTypedQuitIntentCompatibility(t *testing.T) {
	legacy, err := ReadQuitIntent(nil)
	if err != nil || legacy.Kind != QuitIntentDirect {
		t.Fatalf("legacy intent=%#v err=%v", legacy, err)
	}
	for _, intent := range []QuitIntent{
		{Version: QuitIntentVersion, Kind: QuitIntentDirect},
		{Version: QuitIntentVersion, Kind: QuitIntentCouch, Request: &QuitRequestReference{
			DataDir: "/data", RepoScope: "scope", Tag: "work", Nonce: "nonce-1", Attempt: 2,
		}},
	} {
		raw, err := WriteQuitIntent(intent)
		if err != nil {
			t.Fatal(err)
		}
		got, err := ReadQuitIntent(raw)
		if err != nil || !reflect.DeepEqual(got, intent) {
			t.Fatalf("round trip got=%#v err=%v want=%#v", got, err, intent)
		}
	}
	for _, raw := range [][]byte{
		[]byte(`{"version":1,"kind":"couch"}`),
		[]byte(`{"version":1,"kind":"couch","request":{"data_dir":"relative","repo_scope":"scope","tag":"work","nonce":"n","attempt":1}}`),
		[]byte(`{"version":1,"kind":"direct","unknown":true}`),
	} {
		if _, err := ReadQuitIntent(raw); err == nil {
			t.Fatalf("malformed intent accepted: %s", raw)
		}
	}
}

func TestParseRestartMarker(t *testing.T) {
	got := parseRestartMarker("tag=work\nagent=codex\nnew_session=1\nsession_id=SID-LIVE\n")
	want := RestartMarker{Tag: "work", Agent: "codex", NewSession: true, SessionID: "SID-LIVE"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseRestartMarker = %+v, want %+v", got, want)
	}
	// continue + rename keys, and new_session!=1 → false.
	m := parseRestartMarker("tag=t\nnew_session=0\nrename_to=newt\ncontinue=slug-1\n")
	if m.NewSession || m.RenameTo != "newt" || m.Continue != "slug-1" {
		t.Fatalf("parseRestartMarker mixed = %+v", m)
	}
}

func TestSerializeRestartMarkerCarriesSessionID(t *testing.T) {
	m := RestartMarker{Tag: "demo", Agent: "codex", SessionID: "SID-LIVE"}
	got := parseRestartMarker(serializeRestartMarker(m))
	if got != m {
		t.Fatalf("round-trip = %+v, want %+v", got, m)
	}
}

func TestPlanRestart(t *testing.T) {
	saved := savedConfig{Agent: "claude", Args: []string{"--flag"}, SessionID: "SID-1"}

	// The caller (the loop) resolves the marker's tag/agent + any rename before
	// calling, so planRestart takes the FINAL tag/agent. A missing marker binding
	// means the typed launch is provisional: saved config is stale cache only.
	p := planRestart(RestartMarker{}, "work", "claude", saved)
	if !p.DropConfig || p.ContinueSlug != "" {
		t.Fatalf("alt+n plan flags: %+v", p)
	}
	if p.Args.ForcedTag != "work" || p.Args.Agent != "claude" ||
		!reflect.DeepEqual(p.Args.AgentArgs, []string{"--flag"}) {
		t.Fatalf("alt+n args = %+v", p.Args)
	}

	// A session id captured in the marker is newer than saved config because it
	// was read from the live agent immediately before killing the pane.
	pm := planRestart(RestartMarker{SessionID: "SID-LIVE"}, "work", "codex", savedConfig{Agent: "codex", Args: []string{"--flag"}})
	if !reflect.DeepEqual(pm.Args.AgentArgs, []string{"resume", "SID-LIVE", "--flag"}) {
		t.Fatalf("marker session args = %v", pm.Args.AgentArgs)
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

func TestDecideAutomaticResumeConfig(t *testing.T) {
	tests := []struct {
		name         string
		agent        string
		saved        savedConfig
		sessionValid bool
		want         savedConfig
		quarantine   bool
	}{
		{"valid codex root", "codex", savedConfig{Agent: "codex", Args: []string{"--search"}, SessionID: "ROOT"}, true, savedConfig{Agent: "codex", Args: []string{"--search"}, SessionID: "ROOT"}, false},
		{"invalid codex candidate", "codex", savedConfig{Agent: "codex", Args: []string{"--search"}, SessionID: "SUB"}, false, savedConfig{Agent: "codex", Args: []string{"--search"}}, true},
		{"empty codex session", "codex", savedConfig{Agent: "codex", Args: []string{"--search"}}, false, savedConfig{Agent: "codex", Args: []string{"--search"}}, false},
		{"non-codex unchanged", "claude", savedConfig{Agent: "claude", Args: []string{"--flag"}, SessionID: "STALE"}, false, savedConfig{Agent: "claude", Args: []string{"--flag"}, SessionID: "STALE"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, quarantine := decideAutomaticResumeConfig(tt.agent, tt.saved, tt.sessionValid)
			if !reflect.DeepEqual(got, tt.want) || quarantine != tt.quarantine {
				t.Fatalf("got (%+v, %t), want (%+v, %t)", got, quarantine, tt.want, tt.quarantine)
			}
		})
	}
}
