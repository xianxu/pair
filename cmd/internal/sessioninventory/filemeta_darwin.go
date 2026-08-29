//go:build darwin

package sessioninventory

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func platformFileMetadata(path string, _ os.FileInfo) (fileMetadata, error) {
	var stat unix.Stat_t
	if err := unix.Fstatat(unix.AT_FDCWD, path, &stat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return fileMetadata{}, err
	}
	generation := GenerationToken("")
	if stat.Gen != 0 {
		generation = GenerationToken(fmt.Sprintf("gen:%d", stat.Gen))
	}
	return fileMetadata{
		StableFileID:    StableFileID(fmt.Sprintf("dev:%d:ino:%d", stat.Dev, stat.Ino)),
		GenerationToken: generation,
		MutationToken:   MutationToken(fmt.Sprintf("ctime:%d:%d", stat.Ctim.Sec, stat.Ctim.Nsec)),
		BirthTime:       optionalTime(stat.Btim.Sec, stat.Btim.Nsec),
	}, nil
}
