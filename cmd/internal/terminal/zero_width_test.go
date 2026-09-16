package terminal

import (
	"context"
	"fmt"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"
)

func TestZeroWidthOutputKeepsPresentationAndInputUsable(t *testing.T) {
	cases := []string{"e\u0301", "👩\u200d💻", "❤\ufe0e", "❤\ufe0f", "1\ufe0f\u20e3", "🇺🇸", "\u0301", "\u200d", "\ufe0f", "\ufe0e", "\u200b", "\u200c", "\ufeff", "\u2060", "\u034f", "\u0e31", "\u0651", "\u20dd", "\U000e007f", "A\x1b[31m\u0301", "A\x1b[2G\u0301", "界\x1b[31m\u0301", "A\r\u0301", "A\n\r\u0301", "A\x1b[3G\u0301"}
	for _, input := range cases {
		t.Run(fmt.Sprintf("%q", input), func(t *testing.T) {
			for split := 0; split <= len(input); split++ {
				t.Run(fmt.Sprint(split), func(t *testing.T) {
					parent, child := ttyio.NewFake(), ttyio.NewFake()
					endpoint, err := NewEndpoint("zero-width", Geometry{8, 4}, child)
					if err != nil {
						t.Fatal(err)
					}
					defer endpoint.Close()
					presenter := NewPresenter(parent, CouchAnyMotion)
					defer presenter.Release(context.Background())
					if err := presenter.Select(context.Background(), endpoint, Geometry{8, 5}, make([]Cell, 8)); err != nil {
						t.Fatal(err)
					}
					for _, chunk := range []string{input[:split], input[split:], "Z"} {
						if _, err := endpoint.Feed([]byte(chunk), time.Now()); err != nil {
							t.Fatal(err)
						}
						frame, err := endpoint.Snapshot(time.Now())
						if err != nil {
							t.Fatal(err)
						}
						if err := frame.Validate(); err != nil {
							t.Fatalf("after %q: %v", chunk, err)
						}
						if err := presenter.Present(context.Background(), endpoint); err != nil {
							t.Fatal(err)
						}
						if err := presenter.Flush(context.Background()); err != nil {
							t.Fatal(err)
						}
						if presenter.View().State != Ready {
							t.Fatalf("presentation failed: %+v", presenter.View())
						}
					}
					if err := presenter.Input(context.Background(), uv.KeyPressEvent{Code: 'x', Text: "x"}); err != nil {
						t.Fatal(err)
					}
					if err := endpoint.Flush(context.Background()); err != nil {
						t.Fatal(err)
					}
					if got := string(child.Bytes()); got != "x" {
						t.Fatalf("input=%q", got)
					}
				})
			}
		})
	}
}

func TestStyledRowsZeroWidthPolicyMatchesEndpoint(t *testing.T) {
	for _, tc := range []struct{ input, first, second string }{
		{"\u0301A", "A", " "},
		{"A\x1b[31m\u0301B", "A", "B"},
		{"e\u0301", "e\u0301", " "},
		{"\u200d\ufe0fX", "X", " "},
		{"B\x1b[0m\u200d", "B", " "},
		{"\u0301", " ", " "},
	} {
		t.Run(fmt.Sprintf("%q", tc.input), func(t *testing.T) {
			cells, err := StyledRows(tc.input, 8, 1)
			if err != nil {
				t.Fatal(err)
			}
			if cells[0].Content != tc.first || cells[1].Content != tc.second {
				t.Fatalf("cells=%+v", cells[:2])
			}
			frame := Frame{Geometry: Geometry{8, 1}, Cells: cells}
			if err := frame.Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestZeroWidthUnicodeScalarClassProducesValidFrames(t *testing.T) {
	endpoint, err := NewEndpoint("unicode-class", Geometry{8, 4}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer endpoint.Close()
	checked := 0
	for r := rune(0); r <= utf8.MaxRune; r++ {
		if !utf8.ValidRune(r) || unicode.IsControl(r) {
			continue
		}
		scalar := string(r)
		_, width := ansi.FirstGraphemeCluster(scalar, ansi.GraphemeWidth)
		if width != 0 {
			continue
		}
		checked++
		for _, prefix := range []string{"", "A", "A\x1b[31m"} {
			if _, err := endpoint.Feed([]byte("\x1bc"+prefix+scalar), time.Now()); err != nil {
				t.Fatalf("U+%04X: %v", r, err)
			}
			frame, err := endpoint.Snapshot(time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if err := frame.Validate(); err != nil {
				t.Fatalf("U+%04X after %q: %v", r, prefix, err)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no zero-width scalars checked")
	}
	t.Logf("validated %d zero-width Unicode scalars in three contexts", checked)
}
