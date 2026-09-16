package terminalqualify

import (
	"context"
	"io"
	"strconv"
	"strings"

	vt "github.com/charmbracelet/x/vt"
)

// ResourceCases report actual parser storage, not a claimed heap upper bound.
// Usage.ParserBytes is the backing capacity observed from the owned parser;
// RetainedBytes' engineering estimate is deliberately not the oracle here.
func ResourceCases() []Case {
	return []Case{{ID: "parser-memory-bound", Target: "backend-resource", Capability: "bounded parser backing storage and overflow recovery", Source: "third_party/vt/pair_limits.go", Expected: Observation{"parser-start": "65537", "parser-maximum": "65537", "oversize-title-effects": "0", "recovery": "OK"}, Integration: observeParserStorage}}
}

func observeParserStorage(ctx context.Context) (Observation, error) {
	e, err := vt.NewEmulatorWithLimits(8, 4, vt.DefaultLimits())
	if err != nil {
		return nil, err
	}
	defer e.Close()
	e.SetReplyWriter(io.Discard)
	titles := 0
	e.SetCallbacks(vt.Callbacks{Title: func(string) { titles++ }})
	initial := e.Usage().ParserBytes
	maximum := initial
	block := []byte(strings.Repeat("1", 1024))
	// 2MiB each: OSC, DCS, and a single numeric CSI parameter. Feed fixed small
	// pieces so fixture construction itself cannot disguise an unbounded parser.
	for _, stream := range []struct{ begin, end string }{{"\x1b]2;", "\a"}, {"\x1bP1$r", "\x1b\\"}, {"\x1b[", "m"}} {
		if _, err = e.Write([]byte(stream.begin)); err != nil {
			return nil, err
		}
		for i := 0; i < 2048; i++ {
			if err = ctx.Err(); err != nil {
				return nil, err
			}
			if _, err = e.Write(block); err != nil {
				return nil, err
			}
			maximum = max(maximum, e.Usage().ParserBytes)
		}
		if _, err = e.Write([]byte(stream.end)); err != nil {
			return nil, err
		}
	}
	if _, err = e.Write([]byte("\x1b[0m\x1b[HOK")); err != nil {
		return nil, err
	}
	text := ""
	for x := 0; x < 2; x++ {
		if c := e.CellAt(x, 0); c != nil {
			text += c.Content
		}
	}
	return Observation{"parser-start": strconv.Itoa(initial), "parser-maximum": strconv.Itoa(maximum), "oversize-title-effects": strconv.Itoa(titles), "recovery": text}, nil
}
