package testdisk

import (
	"context"
	"fmt"
	"os"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// Retain the exact self generation for both samples. NativeResourceReader
// fences every read; missing identities and read errors never become zero.
func selfWriteReader() func() (uint64, error) {
	r := procutil.NativeResourceReader{}
	var self procutil.Process
	return func() (uint64, error) {
		ctx := context.Background()
		if self.Identity == "" {
			rows, _, err := r.Table(ctx)
			if err != nil {
				return 0, err
			}
			for _, p := range rows {
				if p.PID == os.Getpid() {
					self = p
					break
				}
			}
			if self.Identity == "" {
				return 0, fmt.Errorf("self process generation missing")
			}
		}
		usage, _, err := r.Usage(ctx, self)
		return usage.DiskWriteBytes, err
	}
}
