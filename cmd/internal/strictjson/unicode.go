package strictjson

import (
	"encoding/json"
	"errors"
	"strconv"
	"unicode/utf8"
)

// ValidateUnicode inspects original JSON string bytes before encoding/json
// can replace malformed UTF-8 or unpaired UTF-16 escapes with U+FFFD.
// Preserving decoders call it only on owned values, leaving unknown values raw.
func ValidateUnicode(raw []byte) error {
	if !json.Valid(raw) {
		return errors.New("invalid JSON value")
	}
	if !utf8.Valid(raw) {
		return errors.New("invalid UTF-8 in JSON string")
	}
	for i := 0; i < len(raw); i++ {
		if raw[i] != '"' {
			continue
		}
		i++
		for i < len(raw) && raw[i] != '"' {
			if raw[i] != '\\' {
				i++
				continue
			}
			if i+1 >= len(raw) {
				return errors.New("truncated JSON escape")
			}
			if raw[i+1] != 'u' {
				i += 2
				continue
			}
			if i+6 > len(raw) {
				return errors.New("truncated Unicode escape")
			}
			unit, err := strconv.ParseUint(string(raw[i+2:i+6]), 16, 16)
			if err != nil {
				return err
			}
			i += 6
			if unit >= 0xdc00 && unit <= 0xdfff {
				return errors.New("unpaired low surrogate in JSON string")
			}
			if unit < 0xd800 || unit > 0xdbff {
				continue
			}
			if i+6 > len(raw) || raw[i] != '\\' || raw[i+1] != 'u' {
				return errors.New("unpaired high surrogate in JSON string")
			}
			low, err := strconv.ParseUint(string(raw[i+2:i+6]), 16, 16)
			if err != nil {
				return err
			}
			if low < 0xdc00 || low > 0xdfff {
				return errors.New("unpaired high surrogate in JSON string")
			}
			i += 6
		}
	}
	return nil
}
