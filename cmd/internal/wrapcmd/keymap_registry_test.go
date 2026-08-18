package wrapcmd

import (
	"bytes"
	"testing"
)

// TestTranslateChunk_AgyKeymap exercises the agy row through
// translateChunk so a typo in the registration table that happens to
// pass the registry test (e.g. swapped fields) also gets caught at
// the translation layer.
func TestTranslateChunk_AgyKeymap(t *testing.T) {
	f := newHarnessSessionFake(t, "agy", true)
	t.Cleanup(f.close)
	f.output("\x1b[10;1H──────────\x1b[11;1H> work\x1b[13;1H──────────\x1b[?25h\x1b[12;3H")
	cases := []struct{ in, want []byte }{
		{[]byte("hi\r"), []byte("hi\n")},                                                 // Enter → newline
		{[]byte("hi\x1b\r"), []byte("hi\r")},                                             // Alt+Enter → send
		{[]byte("a\rb\x1b\r"), []byte("a\nb\r")},                                         // both, same chunk
		{[]byte("hi\x1b\x7f"), []byte("hi\x15")},                                         // Alt+Backspace → Ctrl+U
		{[]byte("\x1b[200~text\rmore\x1b[201~"), []byte("\x1b[200~text\rmore\x1b[201~")}, // paste untouched
	}
	for _, c := range cases {
		got, _, _ := f.proxy.translateChunk(c.in, false)
		if !bytes.Equal(got, c.want) {
			t.Errorf("in=%q: got %q, want %q", c.in, got, c.want)
		}
	}
}
