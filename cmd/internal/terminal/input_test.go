package terminal

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

func decodeParts(t *testing.T, parts ...[]byte) []InputEvent {
	t.Helper()
	var d Decoder
	var events []InputEvent
	for _, part := range parts {
		got, err := d.Feed(part)
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, got...)
	}
	return events
}

func TestInputEverySplitPreservesKeysAndRaw(t *testing.T) {
	cases := []struct {
		wire, canonical string
		code            rune
		mod             uv.KeyMod
	}{
		{"\x00", "\x00", uv.KeySpace, uv.ModCtrl},
		{"\x1b[32;5u", "\x00", uv.KeySpace, uv.ModCtrl},
		{"\x7f", "\x7f", uv.KeyBackspace, 0},
		{"\x1b[127;5u", "\x1b[127;5u", uv.KeyBackspace, uv.ModCtrl},
		{"\r", "\r", uv.KeyEnter, 0},
		{"\x1b[13;5u", "\x1b[13;5u", uv.KeyEnter, uv.ModCtrl},
		{"\x1b[104;5u", "\x1b[104;5u", 'h', uv.ModCtrl},
		{"\x1b[116;4u", "\x1bT", 't', uv.ModAlt | uv.ModShift},
		{"\x1bT", "\x1bT", 't', uv.ModAlt | uv.ModShift},
		{"\x1bOA", "\x1b[A", uv.KeyUp, 0},
		{"\x1b[1;3D", "\x1b[1;3D", uv.KeyLeft, uv.ModAlt},
		{"界", "界", '界', 0},
		{"\x1b�", "\x1b�", '�', uv.ModAlt},
	}
	for _, tc := range cases {
		t.Run(tc.wire, func(t *testing.T) {
			wire := []byte(tc.wire)
			for split := 0; split <= len(wire); split++ {
				got := decodeParts(t, wire[:split], wire[split:])
				if len(got) != 1 {
					t.Fatalf("split %d: %+v", split, got)
				}
				key, ok := got[0].Event.(uv.KeyPressEvent)
				if !ok || key.Code != tc.code || key.Mod != tc.mod {
					t.Fatalf("split %d key %#v", split, got[0].Event)
				}
				if got[0].Reply || string(got[0].Raw) != tc.wire || string(got[0].Canonical) != tc.canonical {
					t.Fatalf("split %d: %+v", split, got[0])
				}
			}
		})
	}
}

func TestInputPasteEscapesRemainOneExactPayload(t *testing.T) {
	payload := "x\x1b[13;5u\x00\x1b]52;c;aA==\a界\xff\u009b201~"
	wire := []byte("\x1b[200~" + payload + "\x1b[201~")
	for split := 0; split <= len(wire); split++ {
		events := decodeParts(t, wire[:split], wire[split:])
		if len(events) != 1 {
			t.Fatalf("split %d events %d", split, len(events))
		}
		p, ok := events[0].Event.(uv.PasteEvent)
		if !ok || p.Content != payload || !bytes.Equal(events[0].Raw, wire) || events[0].Reply {
			t.Fatalf("split %d %+v", split, events)
		}
	}
}

func TestInputRepliesMouseFocusAndKeyEventTypes(t *testing.T) {
	cases := []struct {
		raw   string
		event uv.Event
		reply bool
	}{
		{"\x1b[13;5:2u", uv.KeyPressEvent{Code: uv.KeyEnter, Mod: uv.ModCtrl, IsRepeat: true}, false},
		{"\x1b[13;5:3u", uv.KeyReleaseEvent{Code: uv.KeyEnter, Mod: uv.ModCtrl}, false},
		{"\x1b[<0;3;2m", uv.MouseReleaseEvent(uv.Mouse{X: 2, Y: 1, Button: uv.MouseLeft}), false},
		{"\x1b[<32;3;2M", uv.MouseMotionEvent(uv.Mouse{X: 2, Y: 1, Button: uv.MouseLeft}), false},
		{"\x1b[I", uv.FocusEvent{}, false},
		{"\x1b[O", uv.BlurEvent{}, false},
		{"\x1b[?1;2c", nil, true},
		{"\x1b[12;4R", nil, true},
		{"\x1b[?1u", nil, true},
		{"\x1b[99;42z", nil, true},
		{"\x1b]777;unknown\x1b\\", nil, true},
		{"\x1bP1+r544e=787465726d\x1b\\", nil, true},
	}
	for _, tc := range cases {
		for split := 0; split <= len(tc.raw); split++ {
			got := decodeParts(t, []byte(tc.raw[:split]), []byte(tc.raw[split:]))
			if len(got) != 1 {
				t.Fatalf("%q split %d: %+v", tc.raw, split, got)
			}
			if got[0].Reply != tc.reply || string(got[0].Raw) != tc.raw {
				t.Fatalf("%q split %d: %+v", tc.raw, split, got[0])
			}
			if tc.event != nil && !reflect.DeepEqual(got[0].Event, tc.event) {
				t.Fatalf("%q event %#v want %#v", tc.raw, got[0].Event, tc.event)
			}
			if _, ok := got[0].Event.(uv.KeyReleaseEvent); ok && len(got[0].Canonical) != 0 {
				t.Fatal("release may retrigger policy")
			}
		}
	}
}

func TestInputEscapeTimerDoesNotFlushProtocolsOrUTF8(t *testing.T) {
	var d Decoder
	got, err := d.Feed([]byte{27})
	if err != nil || len(got) != 0 || !d.PendingEscape() {
		t.Fatalf("ESC %+v %v", got, err)
	}
	got, err = d.FlushEscape()
	if err != nil || len(got) != 1 || string(got[0].Canonical) != "\x1b" || d.PendingEscape() {
		t.Fatal("ESC flush")
	}
	for _, partial := range []string{"\x1b[13;", "\x1b]10;abc", "\xe7\x95", "\x1b\xe7\x95"} {
		var d Decoder
		if _, err := d.Feed([]byte(partial)); err != nil {
			t.Fatal(err)
		}
		if d.PendingEscape() {
			t.Fatalf("protocol considered ambiguous ESC: %q", partial)
		}
		if got, err := d.FlushEscape(); err != nil || len(got) != 0 {
			t.Fatalf("flushed %q: %+v %v", partial, got, err)
		}
	}
}

func TestInputConcatenationAndBufferOwnership(t *testing.T) {
	wire := []byte("a界\x1b[A\x1b[13;5:3u\x1b[I\x1b[?1u")
	whole := decodeParts(t, wire)
	parts := make([][]byte, len(wire))
	for i := range wire {
		parts[i] = wire[i : i+1]
	}
	split := decodeParts(t, parts...)
	if !reflect.DeepEqual(whole, split) {
		t.Fatalf("whole/split differ: %#v %#v", whole, split)
	}
	var joined []byte
	for _, event := range whole {
		joined = append(joined, event.Raw...)
	}
	if !bytes.Equal(joined, wire) {
		t.Fatal("lost input bytes")
	}
	wire[0] = 'z'
	if string(whole[0].Raw) != "a" {
		t.Fatal("caller buffer retained")
	}
}

func TestInputBoundsFailExplicitly(t *testing.T) {
	for _, wire := range []string{"\x1b]777;" + strings.Repeat("x", MaxStringBytes), "\x1b[200~" + strings.Repeat("x", MaxPasteBytes+1), "\xff"} {
		var d Decoder
		if _, err := d.Feed([]byte(wire)); err == nil {
			t.Fatalf("unbounded or malformed input accepted: length %d prefix %q", len(wire), wire[:min(10, len(wire))])
		}
		if _, err := d.Feed([]byte("x")); err == nil {
			t.Fatal("failed decoder silently resumed")
		}
	}
}

func TestInputAmbiguousCPRNeverBecomesFunctionKey(t *testing.T) {
	got := decodeParts(t, []byte("\x1b[1;3R"))
	if len(got) == 0 {
		t.Fatal("lost ambiguous report")
	}
	for _, e := range got {
		if !e.Reply || len(e.Canonical) != 0 {
			t.Fatalf("CPR leaked operator event: %+v", e)
		}
	}
}

func TestInputMultiReportKeepsBothEventsAndSingleRaw(t *testing.T) {
	raw := "\x1b[48;24;80;480;800t"
	got := decodeParts(t, []byte(raw))
	if len(got) != 2 || !got[0].Reply || !got[1].Reply {
		t.Fatalf("multi-report %+v", got)
	}
	if !reflect.DeepEqual(got[0].Event, uv.WindowSizeEvent{Width: 80, Height: 24}) || !reflect.DeepEqual(got[1].Event, uv.PixelSizeEvent{Width: 800, Height: 480}) {
		t.Fatalf("wrong report values %+v", got)
	}
	if string(got[0].Raw) != raw || len(got[1].Raw) != 0 {
		t.Fatal("raw report repeated/lost")
	}
}

func TestInputC1StringsAndLegacyMouseAreWhole(t *testing.T) {
	for _, raw := range []string{"\x9d777;ǜ\x9c", "\x1b[M #\""} {
		for split := 0; split <= len(raw); split++ {
			got := decodeParts(t, []byte(raw[:split]), []byte(raw[split:]))
			if len(got) != 1 || string(got[0].Raw) != raw {
				t.Fatalf("%q split %d %+v", raw, split, got)
			}
		}
	}
}

func TestInputRepeatPolicyAndReplacementRune(t *testing.T) {
	got := decodeParts(t, []byte("\x1b[13;5:2u�"))
	if len(got) != 2 || string(got[0].Canonical) != "\x1b[13;5u" || got[1].Reply || string(got[1].Canonical) != "�" {
		t.Fatalf("repeat/text %+v", got)
	}
}

func TestInputExactPasteBoundAndUnicodeSplits(t *testing.T) {
	payload := strings.Repeat("p", MaxPasteBytes)
	got := decodeParts(t, []byte("\x1b[200~"+payload), []byte("\x1b[201~"))
	if len(got) != 1 || got[0].Event.(uv.PasteEvent).Content != payload {
		t.Fatal("exact bound paste rejected")
	}
	raw := []byte("é👩‍💻")
	whole := decodeParts(t, raw)
	for i := 0; i <= len(raw); i++ {
		if got := decodeParts(t, raw[:i], raw[i:]); !reflect.DeepEqual(got, whole) {
			t.Fatalf("unicode split %d differs", i)
		}
	}
}

func FuzzInputFraming(f *testing.F) {
	for _, s := range []string{"hello界", "\x1b[1;3R", "\x1b[200~hello\x1b[201~", "\x1bP1$r0m\x1b\\", "\x1b[<0;1;1M"} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 4096 {
			t.Skip()
		}
		var d Decoder
		for i := range data {
			if _, err := d.Feed(data[i : i+1]); err != nil {
				return
			}
		}
		_, _ = d.FlushEscape()
	})
}

func BenchmarkInputSplitString(b *testing.B) {
	data := []byte("\x1b]777;" + strings.Repeat("x", 32000) + "\x1b\\")
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var d Decoder
		for j := range data {
			if _, err := d.Feed(data[j : j+1]); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func TestDecoderLegacyAltArrowAliases(t *testing.T) {
	for _, raw := range []string{"\x1b[3D", "\x1b[3C"} {
		for cut := 0; cut <= len(raw); cut++ {
			var d Decoder
			a, e := d.Feed([]byte(raw[:cut]))
			if e != nil {
				t.Fatal(e)
			}
			b, e := d.Feed([]byte(raw[cut:]))
			if e != nil {
				t.Fatal(e)
			}
			a = append(a, b...)
			if len(a) != 1 || a[0].Reply {
				t.Fatalf("%q split%d: %+v", raw, cut, a)
			}
			k, ok := a[0].Event.(uv.KeyPressEvent)
			if !ok || k.Mod != uv.ModAlt {
				t.Fatalf("%q: %+v", raw, a[0])
			}
		}
	}
}
