package testcouchbridge

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type Document struct {
	ID   string
	File string
}

// Readiness is the complete, replacement client environment and registered files.
// It is published only after the IO owner has checked the actual service.
type Readiness struct {
	Version     int
	Namespace   string
	Socket      string
	Environment map[string]string
	Documents   []Document
}

func ValidateReadiness(r Readiness) error { _, _, err := validateReadiness(r); return err }

// validateReadiness counts bytes visited before JSON allocation. The bounded raw
// budget also bounds JSON's worst-case escaping allocation by a fixed factor.
func validateReadiness(r Readiness) (int, int, error) {
	size, work := 0, 0
	add := func(s string) bool {
		if len(s) > ReadinessLimit-size {
			return false
		}
		size += len(s) + 2
		work += len(s)
		return size <= ReadinessLimit && utf8.ValidString(s) && !strings.ContainsRune(s, 0)
	}
	bad := func(message string) (int, int, error) {
		return size, work, fmt.Errorf("invalid readiness: %s", message)
	}
	if r.Version != 1 {
		return bad("version must be 1")
	}
	if !add(r.Namespace) || !cleanAbsolute(r.Namespace) {
		return bad("namespace must be a bounded clean absolute path")
	}
	if !add(r.Socket) || !cleanAbsolute(r.Socket) || len(r.Socket) > 103 {
		return bad("socket must be a clean absolute path of at most 103 bytes")
	}
	for key, value := range r.Environment {
		if !add(key) || !add(value) || !environmentName(key) {
			return bad("invalid or oversized environment")
		}
	}
	root := r.Environment["COUCH_ISOLATED_ROOT"]
	if !cleanAbsolute(root) || root == string(filepath.Separator) {
		return bad("private isolated root missing")
	}
	if r.Environment["COUCH_STORE_DIR"] != r.Namespace || !inside(root, r.Namespace) {
		return bad("namespace does not match private store")
	}
	paths := filepath.SplitList(r.Environment["PATH"])
	if len(paths) == 0 || paths[0] != filepath.Join(root, "bin") {
		return bad("PATH must begin with private bin")
	}
	for _, path := range paths {
		if !cleanAbsolute(path) || !inside(root, path) {
			return bad("PATH contains an ambient directory")
		}
	}
	if len(r.Documents) != 2 {
		return bad("exactly two registered documents required")
	}
	expected := []string{filepath.Join(root, "artifacts", "first.md"), filepath.Join(root, "artifacts", "first.side", "second.md")}
	for i, document := range r.Documents {
		if !add(document.ID) || !add(document.File) || !documentID(document.ID) || document.File != expected[i] {
			return bad("invalid registered parent/side document layout")
		}
	}
	if r.Documents[0].ID == r.Documents[1].ID {
		return bad("document IDs must be distinct")
	}
	return size, work, nil
}

func environmentName(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range []byte(s) {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_' || (i > 0 && c >= '0' && c <= '9') {
			continue
		}
		return false
	}
	return true
}

func documentID(s string) bool {
	if len(s) != 32 {
		return false
	}
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'f') || (c >= '0' && c <= '9') {
			continue
		}
		return false
	}
	return true
}

func EncodeReadiness(r Readiness) ([]byte, error) {
	if err := ValidateReadiness(r); err != nil {
		return nil, err
	}
	data, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	if len(data) >= ReadinessLimit {
		return nil, fmt.Errorf("readiness exceeds %d bytes including newline", ReadinessLimit)
	}
	return append(data, '\n'), nil
}
