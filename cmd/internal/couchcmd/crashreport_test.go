package couchcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/crashreport"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// The console owner captures into its own store's crash dir and reports what
// the previous incarnation left there (#397).
func TestInstallCrashReportUsesTheStoreAndReportsThePrevious(t *testing.T) {
	store := t.TempDir()
	dir := crashreport.Dir(store)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	previous := filepath.Join(dir, "20261006T135600Z-111.log")
	if err := os.WriteFile(previous, []byte("panic: boom\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	console := couchtty.New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100}), nil)
	capture := installCrashReport(console, store)
	if capture == nil {
		t.Fatal("no capture installed")
	}
	if _, err := os.Stat(filepath.Join(dir, "20261006T135600Z-111.crash")); err != nil {
		t.Fatalf("previous crash not reported from the store's crash dir: %v", err)
	}
	if err := capture.Close(); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("crash dir after clean close = %v, want only the reported crash", entries)
	}
}
