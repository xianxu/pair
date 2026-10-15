//go:build !darwin && !linux

package procutil

// Identity is unavailable without a native process generation token.
func Identity(string) string { return "" }

func nativeIdentity(int) (string, error) { return "", ErrUnsupported }
