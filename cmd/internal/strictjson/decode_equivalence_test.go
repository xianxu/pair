package strictjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"
)

func FuzzRejectDuplicateKeysEquivalent(f *testing.F) {
	for _, raw := range []string{
		`null`, `true`, `42`, `1e400`, `-1e400`, `1e-4000`, `1e309`, `1.7976931348623157e308`,
		`{"a":1,"a":2}`, `{"a":1,"\u0061":2}`, `{"a":{"x":1},"b":{"x":2}}`,
		`[{"a":1,"a":2}]`, `{"a":[{"x":1,"x":2}]}`, `{} {}`, `{"a":null}`, `"a:b"`,
		`{"\ud800":1,"\ufffd":2}`, `{"\ud83d\ude00":1,"😀":2}`, `{"escaped\"key":1}`,
		"{\"\xff\":1,\"\ufffd\":2}",
	} {
		f.Add([]byte(raw))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		oldErr := legacyRejectDuplicateKeys(raw)
		newErr := rejectDuplicateKeys(raw)
		if (oldErr == nil) != (newErr == nil) {
			t.Fatalf("acceptance differs: legacy=%v current=%v raw=%q", oldErr, newErr, raw)
		}
	})
}

func legacyRejectDuplicateKeys(raw []byte) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := legacyScanValue(dec); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}

//lint:allow json-recursive-scanner — ariadne#590: independent legacy-token fuzz oracle; remove when upstream role correction lands.
func legacyScanValue(dec *json.Decoder) error {
	token, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]struct{}{}
		for dec.More() {
			keyToken, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is not a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seen[key] = struct{}{}
			if err := legacyScanValue(dec); err != nil {
				return err
			}
		}
		_, err = dec.Token()
		return err
	case '[':
		for dec.More() {
			if err := legacyScanValue(dec); err != nil {
				return err
			}
		}
		_, err = dec.Token()
		return err
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
}
