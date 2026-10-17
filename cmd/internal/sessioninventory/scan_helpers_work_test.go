package sessioninventory

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

// Framing cost is bytes actually examined for delimiters, independent of host
// load. The fixture's giant first record must not be searched once per chunk.
type framingBytesRuntime struct {
	Runtime
	body []byte
}

func (r framingBytesRuntime) ReadAt(_ Artifact, offset, limit int64) ([]byte, bool, error) {
	end := min(offset+limit, int64(len(r.body)))
	return r.body[offset:end], end == int64(len(r.body)), nil
}

func TestJSONLFramingSearchWorkIsLinear(t *testing.T) {
	for _, size := range []int{int(readChunkSize), 8 * int(readChunkSize), 30_000_000} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			body := append(bytes.Repeat([]byte(" "), size), []byte("{}\nnext\r\ntail")...)
			searched := 0
			search := func(p []byte, needle byte) int {
				n := bytes.IndexByte(p, needle)
				if n < 0 {
					searched += len(p)
				} else {
					searched += n + 1
				}
				return n
			}
			var lines [][]byte
			var offsets []uint64
			tail, stopped, err := frameJSONLArtifactWithSearch(framingBytesRuntime{body: body}, Artifact{}, unlimitedRecordSize, func(line []byte, offset uint64) bool {
				lines = append(lines, bytes.Clone(line))
				offsets = append(offsets, offset)
				return false
			}, search)
			if err != nil || stopped || string(tail) != "tail" || len(lines) != 2 || !bytes.Equal(lines[0], body[:size+2]) || string(lines[1]) != "next\r" || offsets[0] != 0 || offsets[1] != uint64(size+3) {
				t.Fatalf("framing changed: lines=%d offsets=%v tail=%q stopped=%v err=%v", len(lines), offsets, tail, stopped, err)
			}
			if searched != len(body) {
				t.Fatalf("searched %d bytes for %d input bytes; retained prefixes must not be searched again", searched, len(body))
			}
			t.Logf("input=%d delimiter bytes examined=%d", len(body), searched)
		})
	}
}

func TestJSONLFramingChunkBoundariesAndLimits(t *testing.T) {
	long := bytes.Repeat([]byte("x"), int(readChunkSize)*3+7)
	for _, tc := range []struct {
		name        string
		body        []byte
		limit       int64
		stop        bool
		wantLines   [][]byte
		wantOffsets []uint64
		wantTail    []byte
		wantErr     error
	}{
		{name: "newline ends chunk", body: append(bytes.Repeat([]byte("x"), int(readChunkSize)-1), []byte("\n\nend")...), limit: unlimitedRecordSize, wantLines: [][]byte{bytes.Repeat([]byte("x"), int(readChunkSize)-1), {}}, wantOffsets: []uint64{0, uint64(readChunkSize)}, wantTail: []byte("end")},
		{name: "newline starts chunk", body: append(bytes.Repeat([]byte("x"), int(readChunkSize)), []byte("\nnext\n")...), limit: unlimitedRecordSize, wantLines: [][]byte{bytes.Repeat([]byte("x"), int(readChunkSize)), []byte("next")}, wantOffsets: []uint64{0, uint64(readChunkSize) + 1}},
		{name: "exact chunk tail", body: bytes.Repeat([]byte("x"), int(readChunkSize)), limit: unlimitedRecordSize, wantTail: bytes.Repeat([]byte("x"), int(readChunkSize))},
		{name: "long tail", body: long, limit: unlimitedRecordSize, wantTail: long},
		{name: "limit at newline", body: append(bytes.Clone(long), '\n'), limit: int64(len(long)) - 1, wantErr: ErrReadLimit},
		{name: "limit in tail", body: long, limit: int64(len(long)) - 1, wantErr: ErrReadLimit},
		{name: "exact limit", body: append(bytes.Clone(long), '\n'), limit: int64(len(long)), wantLines: [][]byte{long}, wantOffsets: []uint64{0}},
		{name: "early stop", body: append(append(bytes.Clone(long), '\n'), []byte("ignored\n")...), limit: unlimitedRecordSize, stop: true, wantLines: [][]byte{long}, wantOffsets: []uint64{0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var lines [][]byte
			var offsets []uint64
			tail, stopped, err := frameJSONLArtifact(framingBytesRuntime{body: tc.body}, Artifact{}, tc.limit, func(line []byte, offset uint64) bool {
				lines = append(lines, bytes.Clone(line))
				offsets = append(offsets, offset)
				return tc.stop
			})
			if err != tc.wantErr || stopped != tc.stop || !bytes.Equal(tail, tc.wantTail) || len(lines) != len(tc.wantLines) || !reflect.DeepEqual(offsets, tc.wantOffsets) {
				t.Fatalf("lines=%d offsets=%v tail bytes=%d stopped=%t err=%v", len(lines), offsets, len(tail), stopped, err)
			}
			for i := range lines {
				if !bytes.Equal(lines[i], tc.wantLines[i]) {
					t.Fatalf("record %d differs", i)
				}
			}
		})
	}
}
