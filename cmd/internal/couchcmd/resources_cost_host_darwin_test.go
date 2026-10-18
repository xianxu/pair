//go:build darwin

package couchcmd

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/sys/unix"
)

// hostLoadAverage renders darwin's struct loadavg: three little-endian fixpt_t at
// offsets 0/4/8 over an int64 fscale at offset 16. It names the host state a cost
// measurement was taken in (#644) and is report-only, so an unreadable value says
// so instead of failing a run no gate depends on.
func hostLoadAverage() string {
	raw, err := unix.SysctlRaw("vm.loadavg")
	if err != nil || len(raw) < 24 {
		return "loadavg=unknown"
	}
	fscale := binary.LittleEndian.Uint64(raw[16:])
	if fscale == 0 {
		return "loadavg=unknown"
	}
	fixed := func(i int) float64 { return float64(binary.LittleEndian.Uint32(raw[i*4:])) / float64(fscale) }
	return fmt.Sprintf("loadavg=%.2f %.2f %.2f", fixed(0), fixed(1), fixed(2))
}
