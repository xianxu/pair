//go:build darwin

package vmcmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func nativeInvocations(t *testing.T, d *nativeDriver) [][]string {
	t.Helper()
	root := filepath.Join(d.home, "argv")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("native invocation records: %v", err)
	}
	var calls [][]string
	for _, entry := range entries {
		raw, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if len(raw) == 0 || raw[len(raw)-1] != 0 {
			t.Fatalf("incomplete invocation record %q: %q", entry.Name(), raw)
		}
		fields := bytes.Split(raw[:len(raw)-1], []byte{0})
		argv := make([]string, len(fields))
		for i, field := range fields {
			argv[i] = string(field)
		}
		calls = append(calls, argv)
	}
	return calls
}

func nativeInvocationCount(calls [][]string, name string) int {
	count := 0
	for _, argv := range calls {
		if len(argv) > 0 && argv[0] == name {
			count++
		}
	}
	return count
}

func TestNativeFakeRecordsConcurrentInvocationsWithoutLosingArgumentBoundaries(t *testing.T) {
	d := fakeNativeDriver(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	want := map[string][]string{}
	start := make(chan struct{})
	done := make(chan error, 2)
	for _, marker := range []string{"first", "second"} {
		argv := []string{"list", marker, "a b", "line\nbreak", "", "literal; $(touch /poison)"}
		want[marker] = argv
		go func() { <-start; done <- d.command(ctx, argv...).Run() }()
	}
	close(start)
	for range want {
		if err := <-done; err != nil {
			t.Errorf("native fake: %v", err)
		}
	}
	if t.Failed() {
		return
	}
	calls := nativeInvocations(t, d)
	if len(calls) != len(want) {
		t.Fatalf("got %d records, want %d: %q", len(calls), len(want), calls)
	}
	got := map[string][]string{}
	for _, argv := range calls {
		if len(argv) < 2 || got[argv[1]] != nil {
			t.Fatalf("lost invocation identity: %q", calls)
		}
		got[argv[1]] = argv
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("concurrent argv changed: got %q want %q", got, want)
	}
}
