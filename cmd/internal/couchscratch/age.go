package couchscratch

import (
	"os"
	"time"
)

// ageEligible applies the strict age and ordinary-node cleanup policy.
func ageEligible(mode os.FileMode, mtime, now time.Time) bool {
	return (mode.IsRegular() || mode.IsDir()) && mtime.Before(now.Add(-24*time.Hour))
}
