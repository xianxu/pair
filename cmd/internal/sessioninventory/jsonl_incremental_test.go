package sessioninventory

import (
	"bytes"
	"reflect"
	"testing"
)

func TestIncrementalJSONLRetainsTailAcrossArbitraryChunks(t *testing.T) {
	t.Parallel()
	state := JSONLFrameState{ParserCompleteOffset: 5}
	records, state, err := FrameJSONLSuffix(state, []byte("{\"a\":1}\n{\"b\""), 32)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(records, []FramedJSONLRecord{{Offset: 5, Bytes: []byte(`{"a":1}`)}}) || state.ParserCompleteOffset != 13 || !bytes.Equal(state.IncompleteTail, []byte(`{"b"`)) {
		t.Fatalf("records=%#v state=%#v", records, state)
	}
	records, state, err = FrameJSONLSuffix(state, []byte(":2}\n"), 32)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(records, []FramedJSONLRecord{{Offset: 13, Bytes: []byte(`{"b":2}`)}}) || state.ParserCompleteOffset != 21 || len(state.IncompleteTail) != 0 {
		t.Fatalf("records=%#v state=%#v", records, state)
	}
}

func TestIncrementalJSONLRejectsOversizeWithoutAdvancing(t *testing.T) {
	t.Parallel()
	initial := JSONLFrameState{ParserCompleteOffset: 9, IncompleteTail: []byte("abc")}
	if _, got, err := FrameJSONLSuffix(initial, []byte("def\n"), 5); err != ErrReadLimit || !reflect.DeepEqual(got, initial) {
		t.Fatalf("state=%#v err=%v", got, err)
	}
}

func TestIncrementalJSONLDoesNotAliasCallerBytes(t *testing.T) {
	t.Parallel()
	chunk := []byte("one\ntail")
	records, state, err := FrameJSONLSuffix(JSONLFrameState{}, chunk, 32)
	if err != nil {
		t.Fatal(err)
	}
	chunk[0], chunk[4] = 'X', 'X'
	if string(records[0].Bytes) != "one" || string(state.IncompleteTail) != "tail" {
		t.Fatalf("records=%#v state=%#v", records, state)
	}
}

func FuzzFrameJSONLSuffixChunking(f *testing.F) {
	f.Add([]byte("a\nb\npartial"), uint16(3))
	f.Add([]byte("\n"), uint16(1))
	f.Fuzz(func(t *testing.T, raw []byte, chunkSize uint16) {
		if len(raw) > 1<<20 {
			raw = raw[:1<<20]
		}
		size := int(chunkSize)%257 + 1
		oneRecords, oneState, oneErr := FrameJSONLSuffix(JSONLFrameState{}, raw, 1<<20)
		var chunkRecords []FramedJSONLRecord
		chunkState := JSONLFrameState{}
		var chunkErr error
		for start := 0; start < len(raw); start += size {
			end := min(start+size, len(raw))
			var records []FramedJSONLRecord
			records, chunkState, chunkErr = FrameJSONLSuffix(chunkState, raw[start:end], 1<<20)
			if chunkErr != nil {
				break
			}
			chunkRecords = append(chunkRecords, records...)
		}
		if (oneErr == nil) != (chunkErr == nil) || !reflect.DeepEqual(oneRecords, chunkRecords) || !reflect.DeepEqual(oneState, chunkState) {
			t.Fatalf("one=(%#v,%#v,%v) chunked=(%#v,%#v,%v)", oneRecords, oneState, oneErr, chunkRecords, chunkState, chunkErr)
		}
	})
}
