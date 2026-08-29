//go:build linux

package sessioninventory

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func platformFileMetadata(path string, _ os.FileInfo) (fileMetadata, error) {
	var stat unix.Statx_t
	if err := unix.Statx(unix.AT_FDCWD, path, unix.AT_SYMLINK_NOFOLLOW, unix.STATX_BASIC_STATS|unix.STATX_BTIME, &stat); err != nil {
		return fileMetadata{}, err
	}
	return fileMetadata{
		StableFileID:  StableFileID(fmt.Sprintf("dev:%d:%d:ino:%d", stat.Dev_major, stat.Dev_minor, stat.Ino)),
		MutationToken: MutationToken(fmt.Sprintf("ctime:%d:%d", stat.Ctime.Sec, stat.Ctime.Nsec)),
		BirthTime:     optionalTime(stat.Btime.Sec, int64(stat.Btime.Nsec)),
	}, nil
}
