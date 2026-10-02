//go:build !darwin

package main

import (
	"os"
	"strconv"
	"strings"
)

func parentName(pid int) string {
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/comm")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}
