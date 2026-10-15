//go:build linux

package sessioninventory

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
)

// Descriptor equivalent of platformFileMetadata; AT_EMPTY_PATH binds the statx
// result to this open file rather than a replaceable transcript pathname.
func recoveryDescriptorMetadata(f *os.File) (fileMetadata, error) {
	var st unix.Statx_t
	if err := unix.Statx(int(f.Fd()), "", unix.AT_EMPTY_PATH, unix.STATX_BASIC_STATS, &st); err != nil {
		return fileMetadata{}, err
	}
	return fileMetadata{StableFileID: StableFileID(fmt.Sprintf("dev:%d:%d:ino:%d", st.Dev_major, st.Dev_minor, st.Ino)), MutationToken: MutationToken(fmt.Sprintf("ctime:%d:%d", st.Ctime.Sec, st.Ctime.Nsec))}, nil
}
