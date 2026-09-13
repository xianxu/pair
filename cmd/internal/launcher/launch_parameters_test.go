package launcher

import (
	"reflect"
	"strings"
	"testing"
)

func TestLaunchParameters(t *testing.T) {
	for _, tc := range []struct {
		text string
		want []string
	}{
		{"", []string{}}, {`--model 'a b' "" a\ b '雪' '$HOME'`, []string{"--model", "a b", "", "a b", "雪", "$HOME"}},
		{`a"b c"d 'it'\''s'`, []string{"ab cd", "it's"}},
	} {
		got, err := ParseLaunchParameters(tc.text)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("parse %q = %#v, %v", tc.text, got, err)
		}
	}
	for _, text := range []string{"'oops", `"oops`, "abc\\", "a\x00b"} {
		if _, err := ParseLaunchParameters(text); err == nil {
			t.Errorf("accepted %q", text)
		}
	}
	for _, args := range [][]string{{}, {"", "a b", "a'b", `a\"b`, "雪\n字", "$HOME", "$(echo bad)", "*"}} {
		got, err := ParseLaunchParameters(FormatLaunchParameters(args))
		if err != nil || !reflect.DeepEqual(got, args) {
			t.Fatalf("roundtrip %#v: %#v %v", args, got, err)
		}
	}
}
func TestAgentCommandRoundTripAndStrictness(t *testing.T) {
	want := AgentCommand{Executable: "/some path/agent", Argv: []string{"", "a b", "雪", "$(touch nope)", "*", "\n"}}
	raw, err := EncodeAgentCommand(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeAgentCommand(raw)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("roundtrip: %#v %v", got, err)
	}
	for _, raw := range []string{`{}`, `{"executable":"x","argv":null}`, `{"executable":"x","argv":[],"extra":1}`, `{"executable":"x","argv":[]} {}`, `{"executable":"x","argv":["\u0000"]}`, `{"executable":"x","executable":"y","argv":[]}`} {
		if _, err := DecodeAgentCommand(raw); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

func launchArgsText(t *testing.T, env map[string]string) string {
	t.Helper()
	if env[AgentCommandEnv] == "" {
		return ""
	}
	command, err := DecodeAgentCommand(env[AgentCommandEnv])
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(command.Argv, " ")
}
