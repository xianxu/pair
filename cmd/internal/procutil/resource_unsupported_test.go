//go:build !darwin || !cgo

package procutil

import (
	"context"
	"errors"
	"testing"
)

func TestResourceUnsupported(t *testing.T) {
	r := NativeResourceReader{}
	ctx := context.Background()
	p := Process{PID: 1, Identity: "owned"}
	_, n, err := r.Table(ctx)
	if n != 0 || !errors.Is(err, ErrUnsupported) {
		t.Fatal(n, err)
	}
	_, n, err = r.Usage(ctx, p)
	if n != 0 || !errors.Is(err, ErrUnsupported) {
		t.Fatal(n, err)
	}
	_, n, err = r.Command(ctx, p)
	if n != 0 || !errors.Is(err, ErrUnsupported) {
		t.Fatal(n, err)
	}
	_, n, err = r.SelfCPU(ctx)
	if n != 0 || !errors.Is(err, ErrUnsupported) {
		t.Fatal(n, err)
	}
}
