//go:build !darwin

package vmcmd

import "fmt"

func NewNativeDriver() (VMDriver, error) {
	return nil, fmt.Errorf("local Tart VM runs require macOS")
}
