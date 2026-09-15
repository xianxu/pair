package diagnosticlog

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
)

func parseDarwinEnvironment(b []byte) (map[string]string, error) {
	if len(b) < 4 || len(b) > 2<<20 {
		return nil, errors.New("invalid process environment")
	}
	argc := binary.NativeEndian.Uint32(b[:4])
	b = b[4:]
	if argc == 0 || argc > 100000 {
		return nil, errors.New("invalid process argc")
	}
	end := bytes.IndexByte(b, 0)
	if end < 0 {
		return nil, errors.New("truncated executable path")
	}
	b = b[end+1:]
	for len(b) > 0 && b[0] == 0 {
		b = b[1:]
	}
	for range argc {
		end = bytes.IndexByte(b, 0)
		if end < 0 {
			return nil, errors.New("truncated process arguments")
		}
		b = b[end+1:]
	}
	return parseEnvironment(b)
}
func parseEnvironment(b []byte) (map[string]string, error) {
	env := map[string]string{}
	for len(b) > 0 {
		end := bytes.IndexByte(b, 0)
		if end < 0 {
			return nil, errors.New("truncated environment entry")
		}
		if end == 0 {
			break
		}
		key, value, ok := strings.Cut(string(b[:end]), "=")
		if !ok || key == "" {
			return nil, errors.New("invalid environment entry")
		}
		if _, exists := env[key]; exists {
			return nil, errors.New("duplicate environment entry")
		}
		env[key] = value
		b = b[end+1:]
	}
	return env, nil
}
