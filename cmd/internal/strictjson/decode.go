// Package strictjson provides strict decoding for Couch persisted state and
// Pair-owned boundary records, including address claims and the allowed
// one-shot Couch launch profile; sharing the decoder does not share ownership
// of those records.
package strictjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// Decode rejects duplicate object keys, unknown fields, and trailing values.
// Every reader of one durable representation must call this same decoder so
// an alternate projection cannot accept state its owner rejects.
func Decode(raw []byte, target any) error {
	if err := rejectDuplicateKeys(raw); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
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

// RejectDuplicateKeys is Decode's structural check alone: one JSON value with
// no duplicate object key and nothing trailing. It is for boundary readers of
// a versioned EXTERNAL contract that must accept additive unknown fields (the
// sdlc fleet inventory, pair#367) yet refuse an ambiguous document.
func RejectDuplicateKeys(raw []byte) error { return rejectDuplicateKeys(raw) }

// Validate syntax once using the standard library, then inspect only object
// keys. Decoder.Token allocates and converts every value, even values irrelevant
// to duplicate detection. Keep decoded-key equality (including invalid UTF-8
// replacement) and Token's float64 overflow rejection for the external API.
// RejectDuplicateKeysPreserveNumbers validates retained-raw records without
// converting JSON numbers to float64. Known members still need typed decoding.
// Ordinary Decode/RejectDuplicateKeys retain their historical acceptance.
func RejectDuplicateKeysPreserveNumbers(raw []byte) error {
	return rejectDuplicateKeysMode(raw, true)
}

func rejectDuplicateKeys(raw []byte) error { return rejectDuplicateKeysMode(raw, false) }
func rejectDuplicateKeysMode(raw []byte, preserveNumbers bool) error {
	if !json.Valid(raw) {
		return errors.New("invalid JSON value")
	}
	var stack []map[string]struct{}
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '{':
			stack = append(stack, map[string]struct{}{})
		case '[':
			stack = append(stack, nil)
		case '}', ']':
			stack = stack[:len(stack)-1]
		case '"':
			start := i
			decode := false
			i++
			for raw[i] != '"' {
				if raw[i] == '\\' {
					decode = true
					i++
				} else if raw[i] >= 128 {
					decode = true
				}
				i++
			}
			end := i + 1
			next := end
			for next < len(raw) && (raw[next] == ' ' || raw[next] == '\t' || raw[next] == '\r' || raw[next] == '\n') {
				next++
			}
			if next >= len(raw) || raw[next] != ':' {
				continue
			}
			key := string(raw[start+1 : i])
			if decode {
				if err := json.Unmarshal(raw[start:end], &key); err != nil {
					return err
				}
			}
			seen := stack[len(stack)-1]
			if _, ok := seen[key]; ok {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seen[key] = struct{}{}
		default:
			if raw[i] == '-' || (raw[i] >= '0' && raw[i] <= '9') {
				start := i
				for i+1 < len(raw) && raw[i+1] != ',' && raw[i+1] != '}' && raw[i+1] != ']' && raw[i+1] != ' ' && raw[i+1] != '\t' && raw[i+1] != '\r' && raw[i+1] != '\n' {
					i++
				}
				if !preserveNumbers {
					if _, err := strconv.ParseFloat(string(raw[start:i+1]), 64); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
