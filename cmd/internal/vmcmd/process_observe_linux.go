//go:build linux

package vmcmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func nativeChildTerminal(pid int) (bool, error) {
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return false, err
	}
	text := string(raw)
	open := strings.IndexByte(text, '(')
	close := strings.LastIndexByte(text, ')')
	if open < 1 || close < open || strings.TrimSpace(text[:open]) != strconv.Itoa(pid) {
		return false, fmt.Errorf("owned child identity unavailable")
	}
	fields := strings.Fields(text[close+1:])
	if len(fields) == 0 {
		return false, fmt.Errorf("owned child state unavailable")
	}
	return fields[0] == "Z" || fields[0] == "X", nil
}

func nativeGroupPresent(pid int) (bool, error) {
	err := syscall.Kill(-pid, 0)
	if errors.Is(err, syscall.ESRCH) {
		return false, nil
	}
	if err == nil || errors.Is(err, syscall.EPERM) {
		return true, nil
	}
	return false, err
}
