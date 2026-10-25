package zellijquery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
)

// RetireReadyAfter removes positive readiness evidence only after the caller's
// existing authorized lifecycle operation confirms retirement. It adds no native
// effects or retirement authority: retire must establish absence by its existing
// contract. Missing or invalid evidence never prevents that recovery operation.
// A replacement launch published during retirement keeps its own marker.
func RetireReadyAfter(ctx context.Context, env []string, session string, retire func(context.Context) error) (result error) {
	if retire == nil {
		return errors.New("readiness retirement requires an authorized lifecycle callback")
	}
	env = ChildEnvironment(env)
	store, storeErr := newReadinessStore(env)
	var captured ReadyMarker
	var capturedOK bool
	if storeErr == nil {
		marker, err := store.marker(ctx, session)
		if err == nil && QualifyReadyMarker(marker, store.namespace, session) == nil {
			captured = marker
			capturedOK = true
		}
	}
	if err := retire(ctx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !capturedOK {
		return nil
	}
	lease, err := AcquireExplicitCommand(ctx, env)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, lease.Close()) }()
	current, err := store.marker(ctx, session)
	if err := ctx.Err(); err != nil {
		return err
	}
	// Unknown evidence remains untouched even after successful native recovery.
	if err != nil {
		return nil
	}
	if !current.Identity.Equal(captured.Identity) {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := store.paths.ZellijReadyMarker(store.namespace, session)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	// Persist this generation's removal without treating missing evidence as
	// authority about the native session. The callback already proved absence.
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}
