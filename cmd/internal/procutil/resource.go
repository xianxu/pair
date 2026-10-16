package procutil

import (
	"context"
	"errors"
	"syscall"
)

// ResourceUsage contains cumulative CPU and physical disk writes, plus current RSS.
type ResourceUsage struct{ CPUTimeNS, RSSBytes, DiskWriteBytes uint64 }

// NativeResourceReader is a bounded resource observer. Its zero value is usable.
// Counts include failed native calls; unsupported/canceled operations make none.
type NativeResourceReader struct{ calls resourceCalls }
type resourceCalls interface {
	tableSize() (int, error)
	tableRead(int) ([]Process, error)
	usageRead(int) (ResourceUsage, error)
	identityRead(int) (Process, error)
	commandRead(int) (CommandInfo, error)
	environmentRead(int) ([]byte, error)
	selfCPURead() (uint64, error)
}

func (r NativeResourceReader) backend() resourceCalls {
	if r.calls != nil {
		return r.calls
	}
	return platformResourceCalls()
}

const maxResourceRows = 65536

func (r NativeResourceReader) Table(ctx context.Context) ([]Process, int, error) {
	calls := r.backend()
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	if calls == nil {
		return nil, 0, ErrUnsupported
	}
	n := 0
	for attempt := 0; attempt < 3; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, n, err
		}
		size, err := calls.tableSize()
		n++
		if ctx.Err() != nil {
			return nil, n, ctx.Err()
		}
		if err != nil {
			return nil, n, err
		}
		if size < 0 || size > maxResourceRows {
			return nil, n, errors.New("process table exceeds resource bound")
		}
		rows, err := calls.tableRead(size)
		n++
		if ctx.Err() != nil {
			return nil, n, ctx.Err()
		}
		if errors.Is(err, syscall.ENOMEM) {
			continue
		}
		if err != nil {
			return nil, n, err
		}
		if len(rows) > maxResourceRows {
			return nil, n, errors.New("process table exceeds resource bound")
		}
		return rows, n, nil
	}
	return nil, n, errors.New("process table kept growing")
}
func resourceInput(ctx context.Context, p Process) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.PID <= 0 || p.PID > 1<<31-1 || p.Identity == "" {
		return &IdentityUnavailableError{Err: errors.New("missing process generation")}
	}
	if p.Zombie {
		return syscall.ESRCH
	}
	return nil
}
func resourceFence(ctx context.Context, c resourceCalls, p Process) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	current, err := c.identityRead(p.PID)
	if ctx.Err() != nil {
		return 1, ctx.Err()
	}
	if err != nil {
		return 1, &IdentityUnavailableError{Err: err}
	}
	if current.Identity == "" {
		return 1, &IdentityUnavailableError{Err: errors.New("empty process generation")}
	}
	if current.PPID != p.PPID {
		return 1, &IdentityUnavailableError{Err: errors.New("process parent changed during resource observation")}
	}
	if current.PID != p.PID || current.Identity != p.Identity || current.Zombie {
		return 1, syscall.ESRCH
	}
	return 1, nil
}
func (r NativeResourceReader) Usage(ctx context.Context, p Process) (ResourceUsage, int, error) {
	if err := resourceInput(ctx, p); err != nil {
		return ResourceUsage{}, 0, err
	}
	c := r.backend()
	if c == nil {
		return ResourceUsage{}, 0, ErrUnsupported
	}
	u, err := c.usageRead(p.PID)
	if ctx.Err() != nil {
		return ResourceUsage{}, 1, ctx.Err()
	}
	if err != nil {
		return ResourceUsage{}, 1, err
	}
	n, err := resourceFence(ctx, c, p)
	if err != nil {
		return ResourceUsage{}, 1 + n, err
	}
	return u, 1 + n, nil
}
func (r NativeResourceReader) Command(ctx context.Context, p Process) (CommandInfo, int, error) {
	if err := resourceInput(ctx, p); err != nil {
		return CommandInfo{}, 0, err
	}
	c := r.backend()
	if c == nil {
		return CommandInfo{}, 0, ErrUnsupported
	}
	u, err := c.commandRead(p.PID)
	if ctx.Err() != nil {
		return CommandInfo{}, 1, ctx.Err()
	}
	if err != nil {
		return CommandInfo{}, 1, err
	}
	n, err := resourceFence(ctx, c, p)
	if err != nil {
		return CommandInfo{}, 1 + n, err
	}
	return u, 1 + n, nil
}

// Environment returns the named variables of p's initial environment,
// fenced on p's identity like Command. macOS withholds a platform binary's
// environment (zsh, sleep), which reads as an empty map, not an error.
func (r NativeResourceReader) Environment(ctx context.Context, p Process, keys ...string) (map[string]string, int, error) {
	if err := resourceInput(ctx, p); err != nil {
		return nil, 0, err
	}
	c := r.backend()
	if c == nil {
		return nil, 0, ErrUnsupported
	}
	raw, err := c.environmentRead(p.PID)
	if ctx.Err() != nil {
		return nil, 1, ctx.Err()
	}
	if err != nil {
		return nil, 1, err
	}
	n, err := resourceFence(ctx, c, p)
	if err != nil {
		return nil, 1 + n, err
	}
	return ParseEnvironment(raw, keys...), 1 + n, nil
}

// SelfCPU measures the calling OS thread. Lock the goroutine to its OS thread
// across the measurement interval to attribute only the sampler's CPU cost.
func (r NativeResourceReader) SelfCPU(ctx context.Context) (uint64, int, error) {
	if err := ctx.Err(); err != nil {
		return 0, 0, err
	}
	c := r.backend()
	if c == nil {
		return 0, 0, ErrUnsupported
	}
	v, err := c.selfCPURead()
	if ctx.Err() != nil {
		return 0, 1, ctx.Err()
	}
	return v, 1, err
}
