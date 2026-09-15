package vt

import (
	"bytes"
	uv "github.com/charmbracelet/ultraviolet"
	"testing"
)

func TestLegacyModifiedPrintable(t *testing.T) {
	for _, tc := range []struct {
		k    uv.KeyPressEvent
		want string
	}{
		{uv.KeyPressEvent{Code: 'n', Mod: uv.ModAlt | uv.ModShift}, "\x1bN"},
		{uv.KeyPressEvent{Code: 'n', Text: "ñ", Mod: uv.ModAlt}, "\x1bñ"},
		{uv.KeyPressEvent{Code: '1', ShiftedCode: '!', Mod: uv.ModAlt | uv.ModShift}, "\x1b!"},
		{uv.KeyPressEvent{Code: 'n', Text: "N", Mod: uv.ModShift}, "N"},
		{uv.KeyPressEvent{Code: 'n', Mod: uv.ModCtrl | uv.ModAlt}, "\x1b\x0e"},
	} {
		e := NewEmulator(8, 2)
		var out bytes.Buffer
		e.SetReplyWriter(&out)
		e.SendKey(tc.k)
		if out.String() != tc.want {
			t.Errorf("%+v got%q want%q", tc.k, out.String(), tc.want)
		}
		e.Close()
	}
}
