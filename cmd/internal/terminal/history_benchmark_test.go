package terminal

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// Benchmarks retain one caller publication, matching Presenter ownership. They
// expose the initial bounded full-window clone cost rather than hiding it in IO.
func BenchmarkHistoryPublication(b *testing.B) {
	for _, g := range []Geometry{{80, 24}, {240, 80}} {
		for _, fresh := range []bool{false, true} {
			b.Run(fmt.Sprintf("%dx%d/fresh=%v", g.Cols, g.Rows, fresh), func(b *testing.B) {
				e, err := NewEndpoint("history-bench", g, resourceDiscard{})
				if err != nil {
					b.Fatal(err)
				}
				defer e.Close()
				row := strings.Repeat("x", g.Cols-1) + "\r\n"
				if _, err := e.Feed([]byte(strings.Repeat(row, 2000)), time.Time{}); err != nil {
					b.Fatal(err)
				}
				p, err := e.Publication(time.Time{})
				if err != nil {
					b.Fatal(err)
				}
				b.ReportMetric(float64(len(p.History.Rows)), "history-rows")
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if fresh {
						if _, err := e.Feed([]byte("\rX"), time.Time{}); err != nil {
							b.Fatal(err)
						}
					}
					p, err = e.Publication(time.Time{})
					if err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()
				if len(p.History.Rows) == 0 {
					b.Fatal("empty history")
				}
			})
		}
	}
}
