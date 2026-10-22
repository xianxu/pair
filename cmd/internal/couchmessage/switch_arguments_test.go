package couchmessage

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

func TestSwitchArgumentsPresenceAndLiteralRoundTrip(t *testing.T) {
	for _, argv := range [][]string{nil, {}, {"--model", "  quoted '雪' \" $HOME $(touch nope) `literal` ", "", "line\nnext", "\\path"}} {
		t.Run(string(mustSwitchArgumentsJSON(t, argv)), func(t *testing.T) {
			arguments, err := NewSwitchArguments(argv)
			if err != nil {
				t.Fatal(err)
			}
			if arguments.IsZero() {
				t.Fatal("constructor lost explicit argument presence")
			}
			got, present := arguments.Argv()
			if !present || got == nil || !reflect.DeepEqual(got, append([]string{}, argv...)) {
				t.Fatalf("arguments=%q present=%v", got, present)
			}
			request := leanSwitchRequest()
			request.AgentArgs, request.InterruptFirst = arguments, true
			raw, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(raw, append([]byte(`"AgentArgs":`), mustSwitchArgumentsJSON(t, argv)...)) {
				t.Fatalf("arguments were not a literal array: %s", raw)
			}
			var decoded Request
			if err := strictjson.Decode(raw, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded != request { // The complete wire request must remain comparable.
				t.Fatalf("request round trip changed intent: %s", raw)
			}
		})
	}
	var absent SwitchArguments
	if argv, present := absent.Argv(); !absent.IsZero() || present || argv != nil {
		t.Fatalf("zero arguments invented presence: %q %v", argv, present)
	}
	raw, err := json.Marshal(leanSwitchRequest())
	if err != nil || bytes.Contains(raw, []byte("AgentArgs")) {
		t.Fatalf("absent arguments were encoded: %s %v", raw, err)
	}
	var decoded Request
	if err := strictjson.Decode(raw, &decoded); err != nil || !decoded.AgentArgs.IsZero() {
		t.Fatalf("absent arguments did not survive: %#v %v", decoded.AgentArgs, err)
	}
}

func mustSwitchArgumentsJSON(t *testing.T, argv []string) []byte {
	t.Helper()
	raw, err := json.Marshal(append([]string{}, argv...))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestSwitchArgumentsOwnTheirExactValues(t *testing.T) {
	input := []string{"--model", "original"}
	arguments, err := NewSwitchArguments(input)
	if err != nil {
		t.Fatal(err)
	}
	input[1] = "changed input"
	first, _ := arguments.Argv()
	first[1] = "changed output"
	second, _ := arguments.Argv()
	if !reflect.DeepEqual(second, []string{"--model", "original"}) {
		t.Fatalf("arguments alias caller storage: %q", second)
	}
	var decoded SwitchArguments
	if err := strictjson.Decode([]byte(" [ \"--model\", \"original\" ] "), &decoded); err != nil || decoded != arguments {
		t.Fatalf("insignificant JSON whitespace changed intent: %#v %v", decoded, err)
	}
}

func TestSwitchArgumentsRejectInvalidValuesAndFormattedBounds(t *testing.T) {
	for _, argv := range [][]string{{"bad\x00value"}, {string([]byte{0xff})}, {strings.Repeat("x", couchcore.MaxRepoArgumentBytes+1)}, {strings.Repeat("'", 1024)}} {
		if _, err := NewSwitchArguments(argv); err == nil {
			t.Fatalf("accepted invalid argv of formatted size %d", len(launcher.FormatLaunchParameters(argv)))
		}
	}
	if _, err := NewSwitchArguments([]string{strings.Repeat("x", couchcore.MaxRepoArgumentBytes)}); err != nil {
		t.Fatalf("rejected exact formatted bound: %v", err)
	}
}

func TestSwitchArgumentsRejectMalformedWireValues(t *testing.T) {
	for _, raw := range []string{`null`, `"[]"`, `{}`, `true`, `123`, `[null]`, `[1]`, `[true]`, `[{}]`, `[[]]`, `["x",null]`, `["\u0000"]`, `["unfinished"`, `["ok"] []`, "[\"\xff\"]", `[` + strings.Repeat(" ", 32768) + `]`} {
		t.Run(raw[:min(len(raw), 30)], func(t *testing.T) {
			var request Request
			if err := strictjson.Decode([]byte(`{"AgentArgs":`+raw+`}`), &request); err == nil {
				t.Fatalf("accepted wire value %q", raw)
			}
		})
	}
	for _, raw := range []string{`{"AgentArgs":[],"AgentArgs":[]}`, `{"AgentArgs":[],"Unknown":true}`, `{"AgentArgs":[],"Force":true}`} {
		var request Request
		if err := strictjson.Decode([]byte(raw), &request); err == nil {
			t.Fatalf("accepted malformed envelope %s", raw)
		}
	}
}
