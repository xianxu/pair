package couchtty

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"testing"
)

func TestCopyKeyObservationIgnoresUnrelatedInput(t *testing.T) {
	for _, raw := range []string{"y", "z", "\x1b[121;7u", "\x1b[121;6u", "\x1b[120;5u", "\x1b[I", "\x1b[200~private text\x19\x1b[201~"} {
		var decoder terminal.Decoder
		events, err := decoder.Feed([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		for _, event := range events {
			if observation, ok := copyKeyObservation(event); ok {
				t.Fatalf("unrelated %q observed: %+v", raw, observation)
			}
		}
	}
}

func TestCopyKeyObservationContainsNoInputPayload(t *testing.T) {
	event := terminal.InputEvent{Event: uv.KeyPressEvent{Code: 'y', Mod: uv.ModCtrl, Text: "private"}, Raw: []byte("private raw input")}
	got, ok := copyKeyObservation(event)
	if !ok || got != (CopyKeyObservation{Encoding: "enhanced"}) {
		t.Fatalf("observation=%+v ok=%v", got, ok)
	}
}
