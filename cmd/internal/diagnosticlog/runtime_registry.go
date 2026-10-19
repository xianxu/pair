package diagnosticlog

import (
	"context"
	"errors"
	"os"
	"sync"

	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// WithRuntimeRegistry recognizes protocol participation only from the selected
// root's exact incarnation lease. Open holders still pass through unchanged.
func WithRuntimeRegistry(c *storagegc.Coordinator, native Inspection) Inspection {
	return &runtimeInspection{native: native, read: c.ReadRuntimes, probe: c.Probe}
}

type runtimeInspection struct {
	native Inspection
	read   func() ([]storagegc.ProcessRegistration, error)
	probe  storagegc.ProcessProbe
	cached *runtimeObservation
}
type runtimeObservation struct {
	once     sync.Once
	runtimes []Runtime
	err      error
}

func matchesRuntimeRegistration(entry storagegc.ProcessRegistration, process Registration) bool {
	return entry.Role == storagegc.CouchRuntimeRole && entry.DiagnosticProtocol == storagegc.DiagnosticProtocolVersion &&
		entry.Process.PID == process.PID && entry.Process.Birth != "" && entry.Process.Birth == process.Birth
}

func (r *runtimeInspection) OpenFiles(ctx context.Context, path string) ([]Registration, error) {
	return r.native.OpenFiles(ctx, path)
}

func (r *runtimeInspection) Snapshot(ctx context.Context, paths []string) (Inspection, error) {
	native := r.native
	if provider, ok := native.(SweepSnapshot); ok {
		var err error
		native, err = provider.Snapshot(ctx, paths)
		if err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &runtimeInspection{native: native, read: r.read, probe: r.probe, cached: &runtimeObservation{}}, nil
}

func (r *runtimeInspection) Runtimes(ctx context.Context) ([]Runtime, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.cached == nil {
		return r.observe(ctx)
	}
	r.cached.once.Do(func() { r.cached.runtimes, r.cached.err = r.observe(ctx) })
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return r.cached.runtimes, r.cached.err
}

func (r *runtimeInspection) observe(ctx context.Context) ([]Runtime, error) {
	runtimes, err := r.native.Runtimes(ctx)
	if err != nil {
		return nil, err
	}
	entries, err := r.read()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	leases := make(map[storagegc.ProcessIdentity]storagegc.ProcessRegistration, len(entries))
	for _, entry := range entries {
		if entry.Role == storagegc.CouchRuntimeRole && entry.DiagnosticProtocol == storagegc.DiagnosticProtocolVersion {
			leases[entry.Process] = entry
		}
	}
	observed := append([]Runtime(nil), runtimes...)
	for i := range observed {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		runtime := &observed[i]
		if runtime.Protocol {
			continue
		}
		identity := storagegc.ProcessIdentity{PID: runtime.Process.PID, Birth: runtime.Process.Birth}
		entry, ok := leases[identity]
		if !ok || !matchesRuntimeRegistration(entry, runtime.Process) {
			continue
		}
		// The native observation precedes the metadata read: recheck that the same
		// incarnation still owns the declaration before granting participation.
		if r.probe == nil || r.probe.Inspect(identity) != storagegc.ProcessAlive {
			return nil, ErrUnknownWriters
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		runtime.Protocol = true
	}
	return observed, nil
}
