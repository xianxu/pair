package vt

import "testing"

func TestPairIncrementalClusters(t *testing.T) {
	for _, cluster := range []string{"e\u0301", "👩\u200d💻", "1\ufe0f\u20e3", "❤\ufe0f", "🇺🇸"} {
		for split := 0; split <= len(cluster); split++ {
			e := NewEmulator(4, 2)
			e.WriteString(cluster[:split])
			e.WriteString(cluster[split:])
			if got := e.CellAt(0, 0).Content; got != cluster {
				t.Errorf("%q split %d: %q", cluster, split, got)
			}
			e.Close()
		}
	}
}
func TestPairClusterAtBottomRight(t *testing.T) {
	for _, cluster := range []string{"e\u0301", "❤\ufe0f", "1\ufe0f\u20e3", "👩\u200d💻"} {
		for split := 0; split <= len(cluster); split++ {
			e := NewEmulator(4, 2)
			e.WriteString("\x1b[2;4H")
			e.WriteString(cluster[:split])
			e.WriteString(cluster[split:])
			e.WriteString("X")
			if cluster == "e\u0301" {
				if e.CellAt(3, 0).Content != cluster || e.CellAt(0, 1).Content != "X" {
					t.Fatalf("narrow wrap: %q", e.String())
				}
			} else {
				if e.CellAt(0, 1).Content != cluster || e.CellAt(2, 1).Content != "X" {
					t.Fatalf("wide wrap %q split %d: %q", cluster, split, e.String())
				}
			}
			e.Close()
		}
	}
}
func TestPairRetainedAlt47(t *testing.T) {
	e := NewEmulator(4, 2)
	defer e.Close()
	e.WriteString("A\x1b[?47h\x1b[HX\x1b[?47l\x1b[?47h")
	if e.CellAt(0, 0).Content != "X" {
		t.Fatal("47 did not preserve alternate contents")
	}
	e.WriteString("\x1b[?47l\x1b[?1049h")
	if e.CellAt(0, 0).Content == "X" {
		t.Fatal("1049 did not clear alternate contents")
	}
}
