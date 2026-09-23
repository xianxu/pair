package couchcore

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type ProvisionRequest struct {
	Path     string
	Slot     int
	Remote   string
	Progress io.Writer `json:"-"`
}
type ProvisionResult struct {
	SchemaVersion int    `json:"schema_version"`
	Address       string `json:"address"`
	Path          string `json:"path"`
	RestingBranch string `json:"resting_branch"`
	BaselineSHA   string `json:"baseline_sha"`
	Disposition   string `json:"disposition"`
}

func ParseProvisionRequest(path, slot, remote string) (ProvisionRequest, error) {
	if strings.TrimSpace(path) == "" || strings.ContainsRune(path, 0) || !utf8.ValidString(path) {
		return ProvisionRequest{}, fmt.Errorf("provision: path must be nonempty and valid")
	}
	n, err := strconv.Atoi(slot)
	if err != nil || n <= 0 || strconv.Itoa(n) != slot {
		return ProvisionRequest{}, fmt.Errorf("provision: slot must be a canonical positive integer")
	}
	if !utf8.ValidString(remote) || strings.HasPrefix(remote, "-") || strings.IndexFunc(remote, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return ProvisionRequest{}, fmt.Errorf("provision: invalid remote name")
	}
	return ProvisionRequest{Path: path, Slot: n, Remote: remote}, nil
}
