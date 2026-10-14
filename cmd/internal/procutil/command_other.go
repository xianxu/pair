//go:build !darwin && !linux

package procutil

func nativeCommand(int) (CommandInfo, error) { return CommandInfo{}, ErrUnsupported }
