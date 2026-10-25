package ptychild

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Complete production ingest plus consumer publication/validation. Chunk sizes
// model transport boundaries, not semantic terminal operations. No fake parser.
func BenchmarkChildIngestRedraw(b *testing.B) {
	for _, mixed := range []bool{false, true} {
		kind, line := "ascii", strings.Repeat("status 0123456789 ", 4)
		if mixed {
			kind, line = "mixed", strings.Repeat("e\u0301 界 👩\u200d💻 1\ufe0f\u20e3 ", 4)
		}
		var build strings.Builder
		build.WriteString("\x1b[?2026h")
		for row := 1; row <= 24; row++ {
			fmt.Fprintf(&build, "\x1b[%d;1H%s\x1b[K", row, line)
		}
		build.WriteString("\x1b[?2026l")
		payload := []byte(build.String())
		for _, chunkSize := range []int{64, 4096} {
			b.Run(fmt.Sprintf("%s/chunk%d", kind, chunkSize), func(b *testing.B) {
				child := NewFakeChild(nil)
				defer child.Close()
				b.SetBytes(int64(len(payload)))
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					for off := 0; off < len(payload); off += chunkSize {
						if err := child.ingest(payload[off:min(len(payload), off+chunkSize)]); err != nil {
							b.Fatal(err)
						}
					}
					if err := child.FlushOutput(context.Background()); err != nil {
						b.Fatal(err)
					}
					publication, err := child.Endpoint().Publication(time.Now())
					if err != nil {
						b.Fatal(err)
					}
					if err := publication.Frame.Validate(); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
