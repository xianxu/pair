package sessionwatch

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

const lifecycleReadBudget int64 = 256 << 10

type boundOwnerRuntime struct {
	sessioninventory.Runtime
	cache *sessioninventory.OwnerLaunchCache
}

func (r boundOwnerRuntime) OwnerLaunchCache() *sessioninventory.OwnerLaunchCache { return r.cache }
func (r boundOwnerRuntime) DecodeWork(a sessioninventory.Artifact, n int64) {
	if o, ok := r.Runtime.(interface {
		DecodeWork(sessioninventory.Artifact, int64)
	}); ok {
		o.DecodeWork(a, n)
	}
}

// followBoundCodex never returns to discovery. One compact checkpoint follows
// the authorized root; uncertain publication stops with its durable prefix.
func followBoundCodex(opts Options, rt Runtime, native sessioninventory.Runtime, ledgerPath, journalPath string, owner sessionledger.Owner, pid, identity string, validation sessioninventory.TargetValidation, initial []sessioninventory.NativeEventFact, watermark uint64) error {
	ctx := context.Background()
	native = boundOwnerRuntime{Runtime: native, cache: sessioninventory.NewOwnerLaunchCache()}
	ledger := sessioninventory.Artifact{StorageRoot: native.PairDataRoot().Name, RelativePath: filepath.Base(ledgerPath)}
	sid := validation.State.NativeID
	root := sessioninventory.StableID("node", "codex", sid)
	processCurrent := func() bool { return identity == "" || rt.ProcessIdentity(pid) == identity }
	readCurrent := func() (sessionledger.Current, bool, error) {
		current, ok, _, err := sessioninventory.ReadCurrentLaunchAt(ctx, native, owner, ledger)
		return current, ok, err
	}
	ownerCurrent := func() error {
		current, ok, err := readCurrent()
		if err != nil {
			return err
		}
		if !ok || current.Conflict || current.Launch.Ordinal != opts.LaunchOrdinal || current.Binding == nil || current.Binding.RootNativeID != sid {
			return sessionledger.ErrStaleLaunch
		}
		return nil
	}
	if !processCurrent() {
		return nil
	}
	if err := ownerCurrent(); err != nil {
		return err
	}
	cursor, err := sessioninventory.NewTranscriptCursor(native, validation)
	if err != nil {
		rt.Log(adapt.NearMiss, "bound lifecycle target unavailable: "+err.Error())
		return nil
	}
	generation, path, err := lifecycleArtifactIdentity(validation)
	if err != nil {
		return err
	}
	state := boundFollowState{committed: boundCheckpoint{cursor: cursor, scanner: validation.State}, watermark: watermark}
	guard, ok := rt.LedgerAppender().(interface {
		WithCurrentConfirmationReader(string, sessionledger.Owner, uint64, string, func() (sessionledger.Current, bool, error), func() error) error
	})
	if !ok {
		return errors.New("ledger does not support guarded lifecycle publication")
	}
	publish := func(prepared boundPrepared) error {
		if !processCurrent() {
			return sessionledger.ErrStaleLaunch
		}
		if err := ownerCurrent(); err != nil {
			return err
		}
		state = reduceBoundFollow(state, boundFollowEvent{prepared: &prepared})
		if state.phase != followPublishing {
			return nil
		}
		err := guard.WithCurrentConfirmationReader(ledgerPath, owner, opts.LaunchOrdinal, sid, readCurrent, func() error {
			if !processCurrent() {
				return sessionledger.ErrStaleLaunch
			}
			var err error
			state, err = publishLifecycleBatch(opts, rt, journalPath, state)
			return err
		})
		if err != nil {
			state = reduceBoundFollow(state, boundFollowEvent{failure: err})
		}
		return err
	}
	publications, through := lifecyclePublicationsFromEvents(initial, root, state.watermark, opts.LaunchOrdinal, generation, path)
	if err := publish(boundPrepared{checkpoint: state.committed, publications: publications, watermark: through}); err != nil {
		return err
	}
	// Release all bootstrap history before entering the steady follow loop.
	validation = sessioninventory.TargetValidation{}
	initial = nil
	delay := nextLifecycleDelay(opts.Poll, 0, true)
	for {
		rt.Sleep(delay)
		if !processCurrent() {
			return nil
		}
		if err := ownerCurrent(); err != nil {
			return err
		}
		candidate, batch, err := state.committed.cursor.Next(ctx, native, lifecycleReadBudget)
		if err != nil {
			state = reduceBoundFollow(state, boundFollowEvent{failure: err})
			rt.Log(adapt.NearMiss, "bound lifecycle target unavailable: "+err.Error())
			return nil
		}
		scanner := state.committed.scanner
		var events []sessioninventory.NativeEventFact
		if len(batch.Records) > 0 {
			var bytes int64
			for _, r := range batch.Records {
				bytes += int64(len(r.Bytes))
			}
			if o, ok := native.(interface {
				DecodeWork(sessioninventory.Artifact, int64)
			}); ok {
				o.DecodeWork(batch.Entry.Artifact, bytes)
			}
			scanner, _, err = sessioninventory.ValidateCodexDelta(batch.Entry, &scanner, batch.Records)
			if err != nil || scanner.Disputed {
				rt.Log(adapt.NearMiss, "bound lifecycle scanner rejected transcript")
				return nil
			}
			if o, ok := native.(interface {
				DecodeWork(sessioninventory.Artifact, int64)
			}); ok {
				o.DecodeWork(batch.Entry.Artifact, bytes)
			}
			events, _ = sessioninventory.NativeEventsFromRecords(sessioninventory.AgentCodex, root, batch.Records)
		}
		publications, through = lifecyclePublicationsFromEvents(events, root, state.watermark, opts.LaunchOrdinal, generation, path)
		if err := publish(boundPrepared{checkpoint: boundCheckpoint{cursor: candidate, scanner: scanner}, publications: publications, watermark: through}); err != nil {
			return err
		}
		delay = nextLifecycleDelay(opts.Poll, delay, batch.BytesRead > 0 || batch.Backlog)
	}
}
