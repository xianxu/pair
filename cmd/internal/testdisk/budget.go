// Package testdisk defends ordinary test invocations with counted write limits.
package testdisk

import (
	"fmt"
	"math"
	"strings"
)

const (
	CouchCmd         uint64 = 96 << 20
	CouchTTY         uint64 = 64 << 20
	WorkbenchFixture uint64 = 64 << 20
	ThreadActivity   uint64 = 48 << 20
	TitlePoller      uint64 = 24 << 20
	SessionSync      uint64 = 96 << 20
	CouchCore        uint64 = 128 << 20
)

type WriteBudgetVerdict struct{ Bytes, Limit uint64 }

func CheckBytes(before, after, perRun, repeats uint64) (WriteBudgetVerdict, error) {
	if perRun == 0 || repeats == 0 {
		return WriteBudgetVerdict{}, fmt.Errorf("write budget requires positive cap and repeats")
	}
	if repeats > math.MaxUint64/perRun {
		return WriteBudgetVerdict{}, fmt.Errorf("write allowance overflow")
	}
	result := WriteBudgetVerdict{Limit: perRun * repeats}
	if after < before {
		return result, fmt.Errorf("physical write counter decreased")
	}
	result.Bytes = after - before
	if result.Bytes > result.Limit {
		return result, fmt.Errorf("exceeded: bytes=%d cap=%d", result.Bytes, result.Limit)
	}
	return result, nil
}

// testing repeats the suite for every count/CPU pair. A nonpositive count runs
// no tests, but retains one allowance for invocation setup and cleanup.
func invocationRuns(count int, cpus string) (uint64, error) {
	runs := uint64(max(count, 1))
	cpuCount := uint64(1)
	if cpus != "" {
		cpuCount = uint64(len(strings.Split(cpus, ",")))
	}
	if runs > math.MaxUint64/cpuCount {
		return 0, fmt.Errorf("test repetition overflow")
	}
	return runs * cpuCount, nil
}
