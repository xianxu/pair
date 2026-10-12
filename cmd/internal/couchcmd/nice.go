package couchcmd

import (
	"fmt"
	"strconv"
)

// defaultSlotNice keeps slot wrappers, and the builds and suites they run,
// behind Couch and the operator's terminal on a loaded machine (#498).
const defaultSlotNice = 10

// slotNiceSettings reads COUCH_SLOT_NICE: unset means defaultSlotNice, 0 turns
// the lowering off, and anything outside 0-20 refuses to start.
func slotNiceSettings(getenv func(string) string) (int, error) {
	raw := getenv("COUCH_SLOT_NICE")
	if raw == "" {
		return defaultSlotNice, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 || n > 20 {
		return 0, fmt.Errorf("COUCH_SLOT_NICE must be an integer from 0 to 20, got %q", raw)
	}
	return n, nil
}
