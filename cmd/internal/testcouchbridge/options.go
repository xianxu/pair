package testcouchbridge

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	StartupBudget   = 300 * time.Second
	LifetimeBudget  = 600 * time.Second
	StopBudget      = 15 * time.Second
	ReadinessLimit  = 64 << 10
	DiagnosticLimit = 64 << 10
)

// Options selects a candidate binary and the parent of the owned scratch root.
// Physical path checks belong to the owner, before it creates any resources.
type Options struct{ Couch, ScratchParent string }

func ValidateBridgeOptions(o Options) error {
	if !cleanAbsolute(o.ScratchParent) {
		return fmt.Errorf("scratch parent must be a clean absolute path")
	}
	if o.Couch != "" && !cleanAbsolute(o.Couch) {
		return fmt.Errorf("couch must be a clean absolute path")
	}
	return nil
}

func cleanAbsolute(s string) bool {
	return s != "" && len(s) <= ReadinessLimit && utf8.ValidString(s) && !strings.ContainsRune(s, 0) && filepath.IsAbs(s) && filepath.Clean(s) == s
}

func inside(root, path string) bool {
	return path == root || strings.HasPrefix(path, root+string(filepath.Separator))
}
