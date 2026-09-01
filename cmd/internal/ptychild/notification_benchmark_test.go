package ptychild

import (
	"bytes"
	"testing"

	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

func BenchmarkScreenNotificationSkip(b *testing.B) {
	var screen Screen
	screen.Feed(append([]byte(notifyosc.Prefix), bytes.Repeat([]byte{'x'}, notifyosc.MaxMessageBytes+1)...))
	_ = screen.TakeOutputParts()
	chunk := bytes.Repeat([]byte{'x'}, 4096)
	b.SetBytes(int64(len(chunk)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		screen.Feed(chunk)
		_ = screen.TakeOutputParts()
	}
}
