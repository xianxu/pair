package sessionsync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// decodeEditorialObject validates required presence and nullability separately
// from Go's typed decoder, which otherwise accepts null as a scalar zero.
func decodeEditorialObject(raw []byte, target any, required []string) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	known := strictjson.KnownFields(reflect.TypeOf(target).Elem())
	owned, unknown, err := strictjson.SplitObjectPreserveNumbers(raw, known, nil)
	if err != nil {
		return nil, nil, err
	}
	for _, key := range required {
		if _, ok := owned[key]; !ok {
			return nil, nil, fmt.Errorf("missing JSON member %s", key)
		}
	}
	b, err := strictjson.EncodeObjectPreserveNumbers(owned, nil, known)
	if err != nil {
		return nil, nil, err
	}
	if err = json.Unmarshal(b, target); err != nil {
		return nil, nil, err
	}
	return owned, unknown, nil
}

// encodeEditorialObject preserves raw extension values, including nested
// whitespace and large number tokens. Nested owned objects supply their own
// preserving encodings through overrides instead of json.Marshal compaction.
func encodeEditorialObject(value any, unknown, overrides map[string]json.RawMessage) ([]byte, error) {
	known := strictjson.KnownFields(reflect.TypeOf(value))
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var owned map[string]json.RawMessage
	if err := json.Unmarshal(raw, &owned); err != nil {
		return nil, err
	}
	for k, v := range overrides {
		owned[k] = v
	}
	return strictjson.EncodeObjectPreserveNumbers(owned, unknown, known)
}

// UnmarshalJSON reads a span with strict positional tuples and additive fields.
func (s *Span) UnmarshalJSON(raw []byte) error {
	type plain Span
	var v plain
	owned, unknown, err := decodeEditorialObject(raw, &v, []string{"start", "end", "old", "new"})
	if err != nil {
		return err
	}
	for _, key := range []string{"start", "end"} {
		var tuple []json.RawMessage
		if err := json.Unmarshal(owned[key], &tuple); err != nil || len(tuple) != 2 {
			return fmt.Errorf("%s must have exactly two integers", key)
		}
		for _, n := range tuple {
			var i int
			if bytes.Equal(bytes.TrimSpace(n), []byte("null")) {
				return fmt.Errorf("%s cannot contain null", key)
			}
			if err := json.Unmarshal(n, &i); err != nil {
				return err
			}
		}
	}
	if _, present := owned["presentation"]; present {
		if !validSpanPresentation(v.Presentation) {
			return fmt.Errorf("unknown span presentation %q", v.Presentation)
		}
	}
	*s = Span(v)
	s.unknown = unknown
	return nil
}

func validSpanPresentation(s string) bool { return s == "direct" || s == "highlight" || s == "propose" }

func spanPresentation(s Span, document string) string {
	if s.Presentation != "" {
		return s.Presentation
	}
	if document != "" {
		return document
	}
	return "highlight"
}

// MarshalJSON retains a span's additive metadata.
func (s Span) MarshalJSON() ([]byte, error) {
	type plain Span
	return encodeEditorialObject(plain(s), s.unknown, nil)
}

// UnmarshalJSON reads the round envelope without losing extensions.
func (r *Round) UnmarshalJSON(raw []byte) error {
	type plain Round
	var v plain
	owned, unknown, err := decodeEditorialObject(raw, &v, []string{"base_hash", "result_hash", "at", "spans"})
	if err != nil {
		return err
	}
	*r = Round(v)
	if _, ok := owned["presentation"]; !ok {
		r.Presentation = "highlight"
	}
	r.unknown = unknown
	return nil
}

// UnmarshalJSON validates the enclosing artifact as well as its nested round.
func (a *RoundArtifact) UnmarshalJSON(raw []byte) error {
	type plain RoundArtifact
	var v plain
	_, unknown, err := decodeEditorialObject(raw, &v, []string{"file", "round", "body"})
	if err != nil {
		return err
	}
	if v.File == "" {
		return fmt.Errorf("missing artifact file")
	}
	if b, ok := encodeRound(v.Round); !ok {
		return fmt.Errorf("invalid round")
	} else if _, err := decodeRound(b); err != nil {
		return err
	}
	*a = RoundArtifact(v)
	a.unknown = unknown
	return nil
}

// MarshalJSON preserves extensions at both envelope and round levels.
func (a RoundArtifact) MarshalJSON() ([]byte, error) {
	type plain RoundArtifact
	raw, err := a.Round.MarshalJSON()
	if err != nil {
		return nil, err
	}
	v := plain(a)
	v.Round = Round{}
	return encodeEditorialObject(v, a.unknown, map[string]json.RawMessage{"round": raw})
}

// UnmarshalJSON accepts legacy record omissions while validating owned values.
func (r *roundRecord) UnmarshalJSON(raw []byte) error {
	type plain roundRecord
	var v plain
	owned, unknown, err := decodeEditorialObject(raw, &v, []string{"old", "new"})
	if err != nil {
		return err
	}
	if _, ok := owned["presentation"]; ok && !validSpanPresentation(v.Presentation) {
		return fmt.Errorf("invalid record presentation")
	}
	if v.NewOccurrence < 0 {
		return fmt.Errorf("negative occurrence")
	}
	*r = roundRecord(v)
	r.unknown = unknown
	return nil
}

// MarshalJSON retains additive record fields for resume reconstruction.
func (r roundRecord) MarshalJSON() ([]byte, error) {
	type plain roundRecord
	return encodeEditorialObject(plain(r), r.unknown, nil)
}

// MarshalJSON encodes nested spans through their preserving codec.
func (r Round) MarshalJSON() ([]byte, error) {
	type plain Round
	var spans bytes.Buffer
	spans.WriteByte('[')
	for i, s := range r.Spans {
		if i != 0 {
			spans.WriteByte(',')
		}
		raw, err := s.MarshalJSON()
		if err != nil {
			return nil, err
		}
		spans.Write(raw)
	}
	spans.WriteByte(']')
	v := plain(r)
	v.Spans = nil
	return encodeEditorialObject(v, r.unknown, map[string]json.RawMessage{"spans": spans.Bytes()})
}
