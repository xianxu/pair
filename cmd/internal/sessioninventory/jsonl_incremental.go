package sessioninventory

import "bytes"

// JSONLFrameState owns the unparsed record tail beginning at
// ParserCompleteOffset.
// pair:156-concept pure new final
type JSONLFrameState struct {
	ParserCompleteOffset int64  `json:"parser_complete_offset"`
	IncompleteTail       []byte `json:"incomplete_tail,omitempty"`
}

type FramedJSONLRecord struct {
	Offset int64
	Bytes  []byte
}

// FrameJSONLSuffix splits newly observed bytes without treating an incomplete
// final record as malformed. The returned values never alias caller storage.
func FrameJSONLSuffix(state JSONLFrameState, suffix []byte, recordLimit int64) ([]FramedJSONLRecord, JSONLFrameState, error) {
	original := JSONLFrameState{ParserCompleteOffset: state.ParserCompleteOffset, IncompleteTail: append([]byte(nil), state.IncompleteTail...)}
	if state.ParserCompleteOffset < 0 || recordLimit < 0 || int64(len(state.IncompleteTail)) > recordLimit {
		return nil, original, ErrReadLimit
	}
	pending := make([]byte, 0, len(state.IncompleteTail)+len(suffix))
	pending = append(pending, state.IncompleteTail...)
	pending = append(pending, suffix...)
	offset := state.ParserCompleteOffset
	var records []FramedJSONLRecord
	for {
		newline := bytes.IndexByte(pending, '\n')
		if newline < 0 {
			break
		}
		if int64(newline) > recordLimit {
			return nil, original, ErrReadLimit
		}
		recordBytes := pending[:newline]
		if len(recordBytes) > 0 && recordBytes[len(recordBytes)-1] == '\r' {
			recordBytes = recordBytes[:len(recordBytes)-1]
		}
		records = append(records, FramedJSONLRecord{Offset: offset, Bytes: append([]byte(nil), recordBytes...)})
		consumed := int64(newline + 1)
		offset += consumed
		pending = pending[newline+1:]
	}
	if int64(len(pending)) > recordLimit {
		return nil, original, ErrReadLimit
	}
	return records, JSONLFrameState{ParserCompleteOffset: offset, IncompleteTail: append([]byte(nil), pending...)}, nil
}
