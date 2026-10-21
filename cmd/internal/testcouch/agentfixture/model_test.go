package agentfixture

import (
	"reflect"
	"strings"
	"testing"
)

func TestReduceAgentOnlyAcceptsSubmittedPaste(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			state := AgentState{Agent: agent, Phase: Idle}
			var accepted []string
			for _, part := range []string{"\x1b[20", "0~hello\n", "world\x1b[201~"} {
				var got []string
				var err error
				state, got, err = ReduceAgent(state, []byte(part))
				if err != nil || len(got) != 0 {
					t.Fatal(got, err)
				}
			}
			if state.Draft != "hello\nworld" {
				t.Fatal(state)
			}
			before := state
			state, accepted, err := ReduceAgent(state, []byte{'\r'})
			if err != nil || !reflect.DeepEqual(accepted, []string{"hello\nworld"}) || state.Draft != "" || state.Accepted != 1 {
				t.Fatal(state, accepted, err)
			}
			if before.Draft != "hello\nworld" || before.Accepted != 0 {
				t.Fatal("input mutated")
			}
		})
	}
}
func TestReduceAgentNewlinesAndPicker(t *testing.T) {
	s := AgentState{Agent: "claude", Phase: Idle}
	s, got, err := ReduceAgent(s, []byte("one\\\rtwo\r"))
	if err != nil || !reflect.DeepEqual(got, []string{"one\ntwo"}) {
		t.Fatal(s, got, err)
	}
	s.Phase = Picker
	s, got, err = ReduceAgent(s, []byte("never\r"))
	if err != nil || len(got) != 0 {
		t.Fatal(s, got, err)
	}
	_, _, err = ReduceAgent(AgentState{Agent: "codex", Phase: Idle}, []byte(strings.Repeat("x", MaxInputBytes+1)))
	if err == nil {
		t.Fatal("unbounded draft")
	}
}
func TestValidateCommandAndCorrelation(t *testing.T) {
	valid := Command{ID: "request-1", Kind: Exec, Args: []string{"--actors", "--json"}}
	if err := ValidateCommand(valid); err != nil {
		t.Fatal(err)
	}
	for _, c := range []Command{{ID: "../escape", Kind: Exec}, {ID: "ok", Kind: "unknown"}, {ID: "ok", Kind: SetState, State: "unknown"}, {ID: "ok", Kind: Exec, Args: []string{"x\x00y"}}, {ID: "ok", Kind: Stop, Args: []string{"--actors"}}} {
		if ValidateCommand(c) == nil {
			t.Fatalf("accepted %+v", c)
		}
	}
	for _, raw := range []string{`{"id":"one","kind":"exec","args":[],"unknown":1}`, `{"id":"one","id":"two","kind":"stop"}`, `{"id":"one","kind":"stop"} {}`} {
		if _, err := DecodeCommand([]byte(raw)); err == nil {
			t.Fatal("accepted malformed command")
		}
	}
}
func FuzzReduceAgentBounded(f *testing.F) {
	for _, s := range []string{"hello\r", "\x1b[200~body\r\x1b[201~\r", "\xff\r", "\x1b["} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 2*MaxInputBytes {
			return
		}
		s, accepted, err := ReduceAgent(AgentState{Agent: "codex", Phase: Idle}, []byte(input))
		if err != nil {
			return
		}
		if len(s.Draft) > MaxInputBytes || len(s.Escape) > MaxEscapeBytes {
			t.Fatal("unbounded reducer")
		}
		for _, body := range accepted {
			if len(body) > MaxInputBytes {
				t.Fatal("unbounded acceptance")
			}
		}
	})
}

func FuzzDecodeCommand(f *testing.F) {
	for _, raw := range []string{`{"id":"one","kind":"stop"}`, `{"id":"one","id":"two","kind":"exec"}`, `{"id":"../x","kind":"state","state":"picker"}`} {
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		c, err := DecodeCommand([]byte(raw))
		if err == nil {
			if ValidateCommand(c) != nil {
				t.Fatal("decoder admitted invalid request")
			}
		}
	})
}

func TestTerminalRepliesNeverBecomeAuthoredInput(t *testing.T) {
	s := AgentState{Agent: "codex", Phase: Idle}
	for _, piece := range []string{"\x1b]10;rgb:ffff/", "ffff/ffff\x1b", "\\\x1b[38;120R\x1b[?6u", "real input\r"} {
		var accepted []string
		var err error
		s, accepted, err = ReduceAgent(s, []byte(piece))
		if err != nil {
			t.Fatal(err)
		}
		if len(accepted) > 0 && !reflect.DeepEqual(accepted, []string{"real input"}) {
			t.Fatalf("terminal replies became authored input: %#v", accepted)
		}
	}
	if s.Accepted != 1 || s.Draft != "" {
		t.Fatal(s)
	}
}
