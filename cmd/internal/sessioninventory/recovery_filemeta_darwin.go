//go:build darwin

package sessioninventory

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
)

// Descriptor equivalent of platformFileMetadata: identical token encoding, no
// pathname lookup between opening a recovery chunk and checking its generation.
func recoveryDescriptorMetadata(f *os.File) (fileMetadata, error) {
	var st unix.Stat_t
	if err := unix.Fstat(int(f.Fd()), &st); err != nil {
		return fileMetadata{}, err
	}
	generation := GenerationToken("")
	if st.Gen != 0 {
		generation = GenerationToken(fmt.Sprintf("gen:%d", st.Gen))
	}
	return fileMetadata{StableFileID: StableFileID(fmt.Sprintf("dev:%d:ino:%d", st.Dev, st.Ino)), GenerationToken: generation, MutationToken: MutationToken(fmt.Sprintf("ctime:%d:%d", st.Ctim.Sec, st.Ctim.Nsec))}, nil
}
