//go:build linux

package diagnosticlog

import (
	"errors"
	"io"
	"os"
	"strconv"
)

func processEnvironment(pid int) (map[string]string, error) {
	f, e := os.Open("/proc/" + strconv.Itoa(pid) + "/environ")
	if e != nil {
		return nil, e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, (2<<20)+1))
	if e != nil {
		return nil, e
	}
	if len(b) > 2<<20 {
		return nil, errors.New("oversized process environment")
	}
	return parseEnvironment(b)
}
