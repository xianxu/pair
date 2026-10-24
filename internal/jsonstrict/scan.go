// Package jsonstrict validates JSON structure for boundary-specific decoders.
package jsonstrict

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Options selects the structural policy; its zero value keeps exact key
// equality, float64 range validation and unlimited depth.
type Options struct {
	// FoldKeys uses encoding/json-equivalent Unicode simple-fold equality.
	FoldKeys bool
	// PreserveNumbers accepts opaque number tokens outside float64 range.
	PreserveNumbers bool
	// MaxDepth bounds value depth (root is zero); zero disables the bound.
	MaxDepth int
	// VisitKey sees each decoded key at its object depth, before folding.
	VisitKey func(depth int, key string) error
}

// work counts actual byte inspections and decoded keys in the structural pass.
// json.Valid contributes one additional standard-library pass.
type work struct{ bytes, keys int }

// Scan validates exactly one JSON value without materializing its values.
// Typed decoding, schemas, byte bounds and unknown-field retention belong to callers.
func Scan(raw []byte, opts Options) error { return scan(raw, opts, nil) }
func scan(raw []byte, opts Options, w *work) error {
	if !json.Valid(raw) {
		return errors.New("invalid JSON value")
	}
	read := func(i int) byte {
		if w != nil {
			w.bytes++
		}
		return raw[i]
	}
	var stack []map[string]struct{}
	for i := 0; i < len(raw); i++ {
		// A value starts at the current live container depth. Keys are not values.
		if opts.MaxDepth > 0 && len(stack) > opts.MaxDepth && read(i) != '}' && read(i) != ']' && read(i) != ',' && read(i) != ':' && read(i) != ' ' && read(i) != '\t' && read(i) != '\r' && read(i) != '\n' {
			return fmt.Errorf("JSON nesting exceeds %d levels", opts.MaxDepth)
		}
		switch read(i) {
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
			for read(i) != '"' {
				if read(i) == '\\' {
					decode = true
					i++
				} else if read(i) >= 128 {
					decode = true
				}
				i++
			}
			end := i + 1
			next := end
			for next < len(raw) && (read(next) == ' ' || read(next) == '\t' || read(next) == '\r' || read(next) == '\n') {
				next++
			}
			if next >= len(raw) || read(next) != ':' {
				continue
			}
			key := string(raw[start+1 : i])
			if decode {
				if err := json.Unmarshal(raw[start:end], &key); err != nil {
					return err
				}
			}
			if w != nil {
				w.keys++
			}
			originalKey := key
			if opts.FoldKeys {
				key = foldKey(key)
			}
			seen := stack[len(stack)-1]
			if _, ok := seen[key]; ok {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			if opts.VisitKey != nil {
				if err := opts.VisitKey(len(stack)-1, originalKey); err != nil {
					return err
				}
			}
			seen[key] = struct{}{}
		default:
			if read(i) == '-' || (read(i) >= '0' && read(i) <= '9') {
				start := i
				for i+1 < len(raw) && read(i+1) != ',' && read(i+1) != '}' && read(i+1) != ']' && read(i+1) != ' ' && read(i+1) != '\t' && read(i+1) != '\r' && read(i+1) != '\n' {
					i++
				}
				if !opts.PreserveNumbers {
					if _, err := strconv.ParseFloat(string(raw[start:i+1]), 64); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// foldKey follows encoding/json's Unicode simple-fold field matching.
func foldKey(s string) string {
	return strings.Map(func(r rune) rune {
		least := r
		for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
			if f < least {
				least = f
			}
		}
		return least
	}, s)
}
