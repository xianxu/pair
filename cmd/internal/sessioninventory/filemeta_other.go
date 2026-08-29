//go:build !darwin && !linux

package sessioninventory

import (
	"fmt"
	"os"
)

func platformFileMetadata(_ string, info os.FileInfo) (fileMetadata, error) {
	return fileMetadata{
		StableFileID:  StableFileID(fmt.Sprintf("unsupported:%d", info.Size())),
		MutationToken: MutationToken(fmt.Sprintf("mtime:%d", info.ModTime().UnixNano())),
	}, nil
}
