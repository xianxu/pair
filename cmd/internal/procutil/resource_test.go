package procutil

import (
	"context"
	"errors"
	"syscall"
	"testing"
)

type resourceFake struct {
	sizes      []int
	reads      int
	calls      int
	generation Process
	usage      ResourceUsage
	err        error
}

func (f *resourceFake) tableSize() (int, error) { f.calls++; return f.sizes[f.reads], nil }
func (f *resourceFake) tableRead(n int) ([]Process, error) {
	f.calls++
	f.reads++
	if f.reads < len(f.sizes) {
		return nil, syscall.ENOMEM
	}
	return []Process{f.generation}, f.err
}
func (f *resourceFake) usageRead(int) (ResourceUsage, error) { f.calls++; return f.usage, f.err }
func (f *resourceFake) identityRead(int) (Process, error)    { f.calls++; return f.generation, f.err }
func (f *resourceFake) commandRead(int) (CommandInfo, error) {
	f.calls++
	return CommandInfo{Executable: "owned"}, f.err
}
func (f *resourceFake) environmentRead(int) ([]byte, error) {
	f.calls++
	return []byte("COUCH_THREAD_TAG=t\x00OTHER=x\x00"), f.err
}
func (f *resourceFake) selfCPURead() (uint64, error) { f.calls++; return 42, f.err }
func TestResourceBoundedTable(t *testing.T) {
	for _, tc := range []struct {
		sizes []int
		count int
		fail  bool
	}{{[]int{1}, 2, false}, {[]int{1, 2, 3}, 6, false}, {[]int{1, 2, 3, 4}, 6, true}, {[]int{65537}, 1, true}} {
		f := &resourceFake{sizes: tc.sizes}
		r := NativeResourceReader{calls: f}
		_, n, err := r.Table(context.Background())
		if n != tc.count || f.calls != n || (err != nil) != tc.fail {
			t.Fatalf("n=%d actual=%d err=%v", n, f.calls, err)
		}
	}
}
func TestResourceGenerationFence(t *testing.T) {
	p := Process{PID: 10, Identity: "1.2"}
	for _, generation := range []Process{p, {PID: 10, Identity: "1.2", PPID: 1}, {PID: 10, Identity: "2.3"}, {PID: 10}, {PID: 10, Identity: "1.2", Zombie: true}} {
		f := &resourceFake{generation: generation, usage: ResourceUsage{CPUTimeNS: 10, RSSBytes: 20}}
		r := NativeResourceReader{calls: f}
		u, n, err := r.Usage(context.Background(), p)
		good := generation == p
		if n != 2 || (err == nil) != good || (!good && u != (ResourceUsage{})) {
			t.Fatalf("u=%v n=%d err=%v", u, n, err)
		}
		c, n, err := r.Command(context.Background(), p)
		if n != 2 || (err == nil) != good || (!good && c.Executable != "") {
			t.Fatalf("c=%v n=%d err=%v", c, n, err)
		}
		if generation.Identity == "" && IsProcessGone(err) {
			t.Fatal("empty identity falsely proves disappearance")
		}
	}
}
func TestResourceCancellationAndFailureCounts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &resourceFake{}
	r := NativeResourceReader{calls: f}
	_, n, err := r.Table(ctx)
	if n != 0 || !errors.Is(err, context.Canceled) {
		t.Fatal(n, err)
	}
	_, n, err = r.Usage(ctx, Process{PID: 1, Identity: "x"})
	if n != 0 || !errors.Is(err, context.Canceled) {
		t.Fatal(n, err)
	}
	f.err = syscall.EPERM
	_, n, err = r.Usage(context.Background(), Process{PID: 1, Identity: "x"})
	if n != 1 || !errors.Is(err, syscall.EPERM) {
		t.Fatal(n, err)
	}
	_, n, err = r.SelfCPU(ctx)
	if n != 0 || !errors.Is(err, context.Canceled) {
		t.Fatal(n, err)
	}
}
