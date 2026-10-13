//go:build !darwin && !linux

package sessioninventory

import (
	"errors"
	"os"
)

func recoveryDescriptorMetadata(*os.File) (fileMetadata, error) {
	return fileMetadata{}, errors.New("native recovery requires supported descriptor identity")
}
