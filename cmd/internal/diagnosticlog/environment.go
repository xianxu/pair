package diagnosticlog

import (
	"bytes"
	"errors"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"strings"
)

func parseDarwinEnvironment(b []byte) (map[string]string, error) {
	_, rest, err := procutil.ParseDarwinProcArgs(b)
	if err != nil {
		return nil, err
	}
	return parseEnvironment(rest)
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
