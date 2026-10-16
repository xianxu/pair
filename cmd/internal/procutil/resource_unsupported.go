//go:build !darwin || !cgo

package procutil

func platformResourceCalls() resourceCalls { return nil }
