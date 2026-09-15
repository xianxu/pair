package terminalqualify

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func BenchmarkCandidateFeedSnapshot(b *testing.B) {
	for _, size := range [][2]int{{80, 24}, {240, 80}} {
		b.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(b *testing.B) {
			candidate, err := NewCandidate(size[0], size[1])
			if err != nil {
				b.Fatal(err)
			}
			defer candidate.Close()
			input := "\x1b[H" + strings.Repeat("screen qualification ", size[0]*size[1]/21)
			b.ReportAllocs()
			b.SetBytes(int64(len(input)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := candidate.Execute(context.Background(), []string{input}, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
