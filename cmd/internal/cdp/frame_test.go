package cdp

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

// chunkReader returns one chunk per Read, simulating pipe reads that split
// or coalesce frames.
type chunkReader struct{ chunks [][]byte }

func (r *chunkReader) Read(p []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.chunks[0])
	r.chunks[0] = r.chunks[0][n:]
	if len(r.chunks[0]) == 0 {
		r.chunks = r.chunks[1:]
	}
	return n, nil
}

func readAll(t *testing.T, fr *frameReader) ([]string, error) {
	t.Helper()
	var out []string
	for {
		f, err := fr.next()
		if err != nil {
			return out, err
		}
		out = append(out, string(f))
	}
}

func TestFrameReaderSplitAcrossReads(t *testing.T) {
	r := &chunkReader{chunks: [][]byte{[]byte("{\"id\":1}\x00{\"id\""), []byte(":2}\x00")}}
	got, err := readAll(t, newFrameReader(r, MaxFrame))
	if !errors.Is(err, io.EOF) {
		t.Fatalf("err = %v, want io.EOF", err)
	}
	if want := []string{`{"id":1}`, `{"id":2}`}; strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("frames = %q, want %q", got, want)
	}
}

func TestFrameReaderManyFramesOneRead(t *testing.T) {
	r := &chunkReader{chunks: [][]byte{[]byte("{\"a\":1}\x00{\"b\":2}\x00{\"c\":3}\x00")}}
	got, err := readAll(t, newFrameReader(r, MaxFrame))
	if !errors.Is(err, io.EOF) || len(got) != 3 || got[2] != `{"c":3}` {
		t.Fatalf("frames = %q, err = %v", got, err)
	}
}

func TestFrameReaderTruncatedAtEOF(t *testing.T) {
	_, err := readAll(t, newFrameReader(strings.NewReader("{\"id\":1}\x00{\"id\""), MaxFrame))
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestFrameReaderTooLarge(t *testing.T) {
	big := bytes.Repeat([]byte("x"), 200<<10) // spans several bufio fills
	in := append([]byte("{\"id\":1}\x00"), append(big, 0)...)
	got, err := readAll(t, newFrameReader(bytes.NewReader(in), 100<<10))
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("err = %v, want ErrFrameTooLarge", err)
	}
	if len(got) != 1 {
		t.Fatalf("frames before the oversized one = %q", got)
	}
	// The limit is exact: a frame of exactly max bytes passes.
	ok := append(bytes.Repeat([]byte("y"), 32), 0)
	if f, err := newFrameReader(bytes.NewReader(ok), 32).next(); err != nil || len(f) != 32 {
		t.Fatalf("max-size frame: len %d, err %v", len(f), err)
	}
}

func TestErrorString(t *testing.T) {
	e := &Error{Code: -32000, Message: "No target", Data: "id 7"}
	if got := e.Error(); got != "cdp: No target (-32000): id 7" {
		t.Fatalf("Error() = %q", got)
	}
}

func FuzzFrameReader(f *testing.F) {
	f.Add([]byte("{\"id\":1}\x00{\"method\":\"X.y\"}\x00"))
	f.Add([]byte("{\"id\":1}\x00{\"id\""))
	f.Add([]byte("\x00\x00"))
	f.Add(append(bytes.Repeat([]byte("z"), 300), 0))
	f.Fuzz(func(t *testing.T, data []byte) {
		fr := newFrameReader(iotest.HalfReader(bytes.NewReader(data)), 128)
		var consumed []byte
		for {
			frame, err := fr.next()
			if err != nil {
				break
			}
			if len(frame) > 128 || bytes.IndexByte(frame, 0) >= 0 {
				t.Fatalf("bad frame %q", frame)
			}
			consumed = append(consumed, frame...)
			consumed = append(consumed, 0)
			if !bytes.HasPrefix(data, consumed) {
				t.Fatalf("frames %q are not a prefix of input %q", consumed, data)
			}
		}
	})
}
