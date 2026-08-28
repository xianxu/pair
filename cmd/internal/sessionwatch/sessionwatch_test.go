package sessionwatch

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSupportsEveryInventoryAgent(t *testing.T) {
	for _, agent := range []string{"claude", "codex", "agy", "muse"} {
		if !SupportsAgent(agent) {
			t.Fatalf("%s is not watchable", agent)
		}
	}
	if SupportsAgent("unknown") {
		t.Fatal("unknown agent is watchable")
	}
}

func TestStripResumeArgsRemovesCanonicalResumeBindings(t *testing.T) {
	tests := []struct {
		agent string
		args  []string
		want  []string
	}{
		{agent: "codex", args: []string{"resume", "abc", "--no-alt-screen"}, want: []string{"--no-alt-screen"}},
		{agent: "muse", args: []string{"resume", "abc", "--model", "x"}, want: []string{"--model", "x"}},
		{agent: "agy", args: []string{"--model", "x", "--resume", "abc", "--flag"}, want: []string{"--model", "x", "--flag"}},
		{agent: "codex", args: []string{"--foo", "bar", "resume"}, want: []string{"--foo", "bar", "resume"}},
	}
	for _, test := range tests {
		if got := StripResumeArgs(test.agent, test.args); strings.Join(got, "\x00") != strings.Join(test.want, "\x00") {
			t.Fatalf("StripResumeArgs(%q, %#v) = %#v, want %#v", test.agent, test.args, got, test.want)
		}
	}
}

func TestConfigJSONUsesStructuredEncoding(t *testing.T) {
	got, err := ConfigJSON(ConfigPayload{Agent: "codex", Args: []string{`say "hi"`, "--flag"}, SessionID: "019eff64-6ceb-7e72-9d41-a735a97029ac"})
	if err != nil {
		t.Fatal(err)
	}
	var decoded ConfigPayload
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("invalid JSON %q: %v", got, err)
	}
	if decoded.Agent != "codex" || decoded.SessionID == "" || len(decoded.Args) != 2 || decoded.Args[0] != `say "hi"` {
		t.Fatalf("decoded payload = %+v", decoded)
	}
}
