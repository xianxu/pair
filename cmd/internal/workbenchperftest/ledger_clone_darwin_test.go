//go:build darwin && cgo

package workbenchperftest

import (
	"os"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// This is a byte ceiling, independent of host load. Sync each result so the
// native after sample includes the test's dirty pages before it returns.
func TestFixturePaddingPhysicalWriteBudget(t *testing.T) {
	r := procutil.NativeResourceReader{}
	rows, _, err := r.Table(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var self procutil.Process
	for _, p := range rows {
		if p.PID == os.Getpid() {
			self = p
			break
		}
	}
	before, _, err := r.Usage(t.Context(), self)
	if err != nil {
		t.Fatal(err)
	}
	f := NewFixture(t, 8, 1<<20)
	for _, owner := range f.Owners {
		file, err := os.OpenFile(owner.Ledger, os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		syncErr, closeErr := file.Sync(), file.Close()
		if syncErr != nil || closeErr != nil {
			t.Fatalf("sync=%v close=%v", syncErr, closeErr)
		}
	}
	after, _, err := r.Usage(t.Context(), self)
	if err != nil {
		t.Fatal(err)
	}
	if after.DiskWriteBytes < before.DiskWriteBytes {
		t.Fatal("physical write counter decreased")
	}
	const capBytes = 2 << 20
	delta := after.DiskWriteBytes - before.DiskWriteBytes
	t.Logf("fixture physical bytes=%d cap=%d", delta, capBytes)
	if delta > capBytes {
		t.Fatalf("fixture wrote %d physical bytes, cap %d: padding must be seeded once", delta, capBytes)
	}
}
