package couchcore

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/orientation"
)

type ContinuationSource struct {
	Agent         string
	Session       string
	LaunchOrdinal uint64
	// ExpectedDigest binds transport to the bytes validated by the writer or
	// launcher. It is not part of the running source generation identity.
	ExpectedDigest string
}
type ContinuationStatus struct {
	Address   ThreadAddress    `json:"address"`
	RequestID string           `json:"request_id"`
	Phase     checkpoint.Phase `json:"phase"`
	Attempt   string           `json:"attempt,omitempty"`
	Agent     string           `json:"agent"`
	Failure   string           `json:"failure,omitempty"`
}
type ContinuationResult struct {
	SourceReattached bool
	Status           ContinuationStatus
	Record           ActorRecord
	Handle           Handle
	Orientation      *orientation.Request
}

func (r ContinuationResult) Started() (StartResult, bool) {
	return StartResult{Record: r.Record, Handle: r.Handle}, r.Handle != nil
}
func continuationStatus(record ThreadRecord) *ContinuationStatus {
	r := record.Continuation
	if r == nil {
		return nil
	}
	return &ContinuationStatus{Address: record.Address, RequestID: r.ID, Phase: r.Phase, Attempt: r.Attempt, Agent: r.Source.Agent, Failure: r.Failure}
}
func (c *Couch) ContinuationRequests(ctx context.Context, addresses []ThreadAddress) ([]ContinuationStatus, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	out := make([]ContinuationStatus, 0, len(addresses))
	var failures []error
	seen := map[ThreadAddress]bool{}
	for _, a := range addresses {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if seen[a] {
			continue
		}
		seen[a] = true
		r, err := c.Threads.GetThread(a)
		if errors.Is(err, ErrThreadNotFound) {
			continue
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("continuation %s: %w", a.Tag, err))
			continue
		}
		if status := continuationStatus(r); status != nil {
			out = append(out, *status)
		}
	}
	return out, errors.Join(failures...)
}
func (c *Couch) RequestContinuation(ctx context.Context, address ThreadAddress, source ContinuationSource, path string) (ContinuationStatus, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateThreadAddress(address); err != nil {
		return ContinuationStatus{}, err
	}
	if c == nil || c.Threads == nil || c.ContinuationSource == nil {
		return ContinuationStatus{}, errors.New("continuation source reader unavailable")
	}
	cp, err := checkpoint.ReadFile(path)
	if err != nil {
		return ContinuationStatus{}, err
	}
	if source.ExpectedDigest != "" && source.ExpectedDigest != cp.Digest {
		return ContinuationStatus{}, errors.New("checkpoint changed after writer validation; save and submit the exact checkpoint again")
	}
	if !launcher.IsSupportedAgent(source.Agent) || source.Agent != cp.Agent() || source.Session == "" || source.LaunchOrdinal == 0 {
		return ContinuationStatus{}, errors.New("checkpoint and source identity must match")
	}
	for tries := 0; tries < 8; tries++ {
		if err := ctx.Err(); err != nil {
			return ContinuationStatus{}, err
		}
		current, err := c.ContinuationSource(ctx, address)
		if err != nil {
			return ContinuationStatus{}, err
		}
		if !sameContinuationSource(current, source) {
			return ContinuationStatus{}, errors.New("continuation source generation is obsolete")
		}
		record, err := c.Threads.GetThread(address)
		if err != nil {
			return ContinuationStatus{}, err
		}
		id := checkpoint.RequestID(address.RepoScope, string(address.Tag), source.LaunchOrdinal, cp.Digest)
		if record.Continuation != nil && record.Continuation.ID == id {
			return *continuationStatus(record), nil
		}
		incarnation, err := soleParkableIncarnation(record)
		if err != nil {
			return ContinuationStatus{}, err
		}
		if len(record.Incarnations) != 1 || incarnation.State != IncarnationLive || record.Park != nil || observeExactProcess(c.Proc, ProcessIdentity{PID: incarnation.PID, Identity: incarnation.Identity}) != Live {
			return ContinuationStatus{}, errors.New("source helper is not provably live")
		}
		request := checkpoint.Request{Version: checkpoint.Version, ID: id, Checkpoint: cp, Source: checkpoint.Source{Agent: source.Agent, Session: source.Session, LaunchOrdinal: source.LaunchOrdinal, Helper: checkpoint.Process{PID: incarnation.PID, Identity: incarnation.Identity}}, CreatedAt: c.Clock.Now(), Phase: checkpoint.Pending}
		published, err := c.Threads.PublishContinuation(address, record.Revision, request)
		var stale *ThreadRevisionError
		if errors.As(err, &stale) {
			continue
		}
		if err != nil {
			return ContinuationStatus{}, err
		}
		return *continuationStatus(published), nil
	}
	return ContinuationStatus{}, errors.New("continuation publication raced with thread updates; retry")
}

func sameContinuationSource(a, b ContinuationSource) bool {
	return a.Agent == b.Agent && a.Session == b.Session && a.LaunchOrdinal == b.LaunchOrdinal
}
func (c *Couch) requestRecord(address ThreadAddress, id string) (ThreadRecord, error) {
	r, err := c.Threads.GetThread(address)
	if err != nil {
		return r, err
	}
	if r.Continuation == nil {
		return r, errors.New("thread has no continuation request")
	}
	if id != "" && r.Continuation.ID != id {
		return r, errors.New("obsolete continuation request")
	}
	return r, nil
}
func (c *Couch) advanceContinuation(address ThreadAddress, event checkpoint.Event) (ThreadRecord, error) {
	return c.writeRequestRecord(address, event.RequestID, func(r ThreadRecord) (ThreadRecord, error) {
		return c.Threads.AdvanceContinuation(address, r.Revision, event)
	})
}

// writeRequestRecord reads the exact request's record and applies one store
// transition to it, re-reading on a stale revision: the one loop every
// request-scoped transition shares.
func (c *Couch) writeRequestRecord(address ThreadAddress, id string, write func(ThreadRecord) (ThreadRecord, error)) (ThreadRecord, error) {
	for tries := 0; tries < 8; tries++ {
		r, err := c.requestRecord(address, id)
		if err != nil {
			return r, err
		}
		next, err := write(r)
		var stale *ThreadRevisionError
		if errors.As(err, &stale) {
			continue
		}
		return next, err
	}
	return ThreadRecord{}, errors.New("continuation transition raced with thread updates")
}
func (c *Couch) failContinuation(record ThreadRecord, cause error) (ContinuationResult, error) {
	r := record.Continuation
	next, err := c.advanceContinuation(record.Address, checkpoint.Event{Kind: checkpoint.Fail, RequestID: r.ID, Attempt: r.Attempt, Failure: cause.Error()})
	if err == nil {
		record = next
	}
	return ContinuationResult{Status: *continuationStatus(record)}, errors.Join(cause, err)
}
func (c *Couch) Continue(ctx context.Context, address ThreadAddress, id string) (ContinuationResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return ContinuationResult{}, err
	}
	record, err := c.requestRecord(address, id)
	if err != nil {
		return ContinuationResult{}, err
	}
	if record.Continuation.Phase == checkpoint.Complete {
		return ContinuationResult{Status: *continuationStatus(record)}, nil
	}
	if record.Continuation.Phase == checkpoint.Failed {
		return ContinuationResult{Status: *continuationStatus(record)}, continuationGuard(record)
	}
	if record.Continuation.Phase == checkpoint.Pending {
		attempt, err := allocateStartNonce(c.Entropy)
		if err != nil {
			return ContinuationResult{}, err
		}
		record, err = c.Threads.AdvanceContinuation(address, record.Revision, checkpoint.Event{Kind: checkpoint.Begin, RequestID: record.Continuation.ID, Attempt: attempt})
		if err != nil {
			return ContinuationResult{}, err
		}
	}
	return c.executeContinuation(ctx, record)
}
func (c *Couch) executeContinuation(ctx context.Context, record ThreadRecord) (ContinuationResult, error) {
	request := record.Continuation
	if c.FreshRegistration == nil || c.PairLifecycle == nil || c.ContinuationSource == nil {
		return c.failContinuation(record, errors.New("continuation lifecycle services unavailable"))
	}
	// A ready receipt belongs to a target even if Couch crashed before recording
	// Target. Never infer target absence from a missing local callback.
	registered, err := c.FreshRegistration(ctx, record.Address, request.Source.Agent, request.Attempt)
	if err != nil {
		return c.failContinuation(record, err)
	}
	if registered || request.Target != nil {
		return c.observeContinuationTarget(ctx, record, registered)
	}
	profile := record.LatestLaunchProfile
	if profile == nil || profile.Agent != request.Source.Agent {
		return c.failContinuation(record, errors.New("continuation launch profile no longer matches source"))
	}
	argv := launcher.FreshAgentArgs(profile.Argv)
	if err := launcher.ValidateFreshAgentArgs(profile.Agent, argv); err != nil {
		return c.failContinuation(record, err)
	}
	raw, err := launcher.BuildCouchFreshLaunchProfile(string(record.Address.Tag), profile.Agent, argv, string(AgentSourcePath), string(ArgvSourcePath))
	if err != nil {
		return c.failContinuation(record, err)
	}
	for _, inc := range record.Incarnations {
		if inc.Start != nil && inc.Start.Nonce == request.Attempt {
			return c.failContinuation(record, errors.New("continuation target start is unresolved; inspect its exact helper and session before retry"))
		}
	}
	if record.Park != nil {
		recovered, err := c.PairLifecycle.Recover(ctx, record.Address)
		if err != nil {
			return c.failContinuation(record, err)
		}
		record = recovered.Thread
	}
	if record.VerifiedPark == nil && !continuationSourceParked(record) && request.SourceAbsence == nil {
		current, err := c.ContinuationSource(ctx, record.Address)
		if err != nil {
			return c.failContinuation(record, err)
		}
		if !sameContinuationSource(current, ContinuationSource{Agent: request.Source.Agent, Session: request.Source.Session, LaunchOrdinal: request.Source.LaunchOrdinal}) {
			return c.failContinuation(record, errors.New("continuation source generation changed before park"))
		}
		var attachedActor ActorRecord
		var attachedHandle Handle
		record, attachedActor, attachedHandle, err = c.ensureContinuationAttached(ctx, record, false)
		if err != nil {
			return c.failContinuation(record, err)
		}
		inc, err := soleParkableIncarnation(record)
		if err != nil {
			return c.failContinuation(record, err)
		}
		if !c.ownsContinuationHelper(record.Address, inc) {
			return c.failContinuation(record, errors.New("continuation source belongs to another live Couch owner"))
		}
		if inc.PID != request.Source.Helper.PID || inc.Identity != request.Source.Helper.Identity {
			updated, err := c.advanceContinuation(record.Address, checkpoint.Event{Kind: checkpoint.RefreshSource, RequestID: request.ID, Attempt: request.Attempt, Helper: checkpoint.Process{PID: inc.PID, Identity: inc.Identity}})
			if err != nil {
				if attachedHandle != nil {
					err = errors.Join(err, c.AbortStarted(StartResult{Record: attachedActor, Handle: attachedHandle}, err))
				}
				return c.failContinuation(record, err)
			}
			record = updated
		}
		if attachedHandle != nil {
			// Hand every created terminal to Console before any subsequent
			// lifecycle step. Its next poll continues the already-running request.
			return ContinuationResult{Status: *continuationStatus(record), Record: attachedActor, Handle: attachedHandle, SourceReattached: true}, nil
		}
		// The CAS in ParkExpected binds teardown to the source proof just read.
		current, err = c.ContinuationSource(ctx, record.Address)
		if err != nil || current.LaunchOrdinal != request.Source.LaunchOrdinal || current.Agent != request.Source.Agent || current.Session != request.Source.Session {
			if err == nil {
				err = errors.New("continuation source changed immediately before park")
			}
			return c.failContinuation(record, err)
		}
		parked, err := c.PairLifecycle.ParkExpected(ctx, record.Address, record.Revision)
		if err != nil {
			return c.failContinuation(record, err)
		}
		record = parked.Thread
	}
	if request.SourcePark == "" && request.SourceAbsence == nil {
		if record.VerifiedPark == nil {
			return c.failContinuation(record, errors.New("continuation source has no verified park receipt"))
		}
		record, err = c.advanceContinuation(record.Address, checkpoint.Event{Kind: checkpoint.SourceParked, RequestID: request.ID, Attempt: request.Attempt, ParkNonce: record.VerifiedPark.Identity.Nonce})
		if err != nil {
			return ContinuationResult{}, err
		}
		request = record.Continuation
	}
	if err := ctx.Err(); err != nil {
		return c.failContinuation(record, err)
	}
	path, err := c.materializeContinuation(record)
	if err != nil {
		return c.failContinuation(record, err)
	}
	owner, err := c.Proc.Current()
	if err != nil {
		return c.failContinuation(record, err)
	}
	repoIdentity, err := c.resolveRepoIdentity(ctx, record.WorkingPath)
	if err != nil {
		return c.failContinuation(record, err)
	}
	if request.SourceAbsence != nil {
		if err := c.verifyAbsentContinuation(ctx, record); err != nil {
			return c.failContinuation(record, err)
		}
	} else if err := c.verifyContinuationGeneration(ctx, record); err != nil {
		return c.failContinuation(record, err)
	}
	claimed, err := c.Threads.CommitStartClaim(record.Address, record.Revision, repoIdentity, c.Clock.Now(), StartEvent{Kind: StartClaimed, Nonce: request.Attempt, Owner: SupervisorOwner{PID: owner.PID, Identity: owner.Identity}, Profile: profile, Shape: StartFreshExisting})
	if err != nil {
		return c.failContinuation(record, err)
	}
	var authorityErr error
	if request.SourceAbsence != nil {
		authorityErr = c.verifyAbsentContinuation(ctx, claimed)
	} else {
		authorityErr = c.verifyContinuationGeneration(ctx, claimed)
	}
	if authorityErr != nil {
		cleanupErr := c.rollbackTrackedStart(claimed, request.Attempt)
		return c.failContinuation(record, errors.Join(authorityErr, cleanupErr))
	}
	orient := orientation.Request{SchemaVersion: 1, Tag: string(record.Address.Tag), Agent: profile.Agent, Attempt: request.Attempt, Body: "Read the saved continuation at " + strconv.Quote(path) + " (SHA-256 " + request.Checkpoint.Digest + "). Resume the task from its NEXT ACTION and preserve the existing Pair thread and prompt history."}
	actor, handle, err := c.launchTrackedThread(trackedThreadLaunch{Context: ctx, Thread: claimed, Nonce: request.Attempt, Args: StartArgs{Worktree: Worktree(record.StartingPath), Cwd: record.WorkingPath, Stack: profile.Agent, ExtraArgs: argv}, StartedAt: c.Clock.Now(), ProfileRaw: raw, Fresh: true, Orientation: &orient})
	if err != nil {
		return c.failContinuation(record, err)
	}
	generation, generationErr := c.continuationTargetGeneration(ctx, record)
	if generationErr != nil {
		cleanupErr := c.AbortStarted(StartResult{Record: actor, Handle: handle}, generationErr)
		return c.failContinuation(record, errors.Join(generationErr, cleanupErr))
	}
	target := checkpoint.Process{PID: actor.PID, Identity: actor.Identity}
	registeredRecord, persistErr := c.advanceContinuation(record.Address, checkpoint.Event{Kind: checkpoint.Registered, At: c.Clock.Now(), RequestID: request.ID, Attempt: request.Attempt, Target: &target, TargetGeneration: generation})
	if persistErr != nil {
		cleanupErr := c.AbortStarted(StartResult{Record: actor, Handle: handle}, persistErr)
		return c.failContinuation(record, errors.Join(persistErr, cleanupErr))
	}
	record = registeredRecord
	return ContinuationResult{Status: *continuationStatus(record), Record: actor, Handle: handle, Orientation: &orient}, persistErr
}
func (c *Couch) ownsContinuationHelper(address ThreadAddress, inc ThreadIncarnation) bool {
	for _, a := range c.reg.Records() {
		if a.Thread == address && a.PID == inc.PID && a.Identity == inc.Identity {
			return true
		}
	}
	return false
}
func continuationGuard(record ThreadRecord) error {
	if r := record.Continuation; r != nil && r.Phase != checkpoint.Complete {
		return fmt.Errorf("continuation %s is %s; %s", r.ID, r.Phase, checkpoint.Exits(r.Phase, string(record.Address.Tag)))
	}
	return nil
}

// withContinuationExits appends the retained request's exits to a refusal that
// request caused. Every refusal of an operation a retained unfinished request
// blocks goes through here or through continuationGuard -- archive, warm
// reattach, recovery -- so none names only one way out (pair#280).
func withContinuationExits(record ThreadRecord, err error) error {
	r := record.Continuation
	if err == nil || r == nil || r.Phase == checkpoint.Complete {
		return err
	}
	return fmt.Errorf("%w; the retained continuation is %s: %s", err, r.Phase, checkpoint.Exits(r.Phase, string(record.Address.Tag)))
}

// ContinuationRefuses is the single statement of continuationGuard's reach:
// the operations it refuses while a thread retains an unfinished request --
// relaunch (relaunch.go), switch-agent (switchagent.go), a cold resume
// (resume.go) and every non-warm start claim (threadstore.go). Park and detach
// never read the request. The switcher filters a failed row's actions through
// this rather than restating the list. TestContinuationRefusesMatchesTheGuard-
// ForEveryRowAction drives relaunch, switch-agent's preview (which SwitchAgent
// re-runs), a cold resume and a cold start claim into the guard -- refused by
// its own words, having written nothing -- and park, detach, name and describe
// to success, through the production dispatcher (#280).
func ContinuationRefuses(operation string) bool {
	switch operation {
	case "relaunch", "switch-agent", "prepare-switch-agent", "resume", "start":
		return true
	}
	return false
}

// DismissContinuation deletes the thread's retained FAILED continuation (#280).
// The operator has decided the thread moved on -- typically by taking over the
// target before automatic orientation finished -- so re-delivering the handoff
// would be wrong. Couch's private checkpoint copy is left for the next publish
// or archive to replace; the repository's checkpoint file is never touched.
func (c *Couch) DismissContinuation(ctx context.Context, address ThreadAddress, id string) (ThreadRecord, error) {
	if ctx != nil && ctx.Err() != nil {
		return ThreadRecord{}, ctx.Err()
	}
	return c.writeRequestRecord(address, id, func(r ThreadRecord) (ThreadRecord, error) {
		return c.Threads.DismissFailedContinuation(address, r.Revision, r.Continuation.ID)
	})
}

func continuationSourceParked(record ThreadRecord) bool {
	r := record.Continuation
	if r == nil || r.SourcePark == "" {
		return false
	}
	for _, park := range record.ParkHistory {
		if park.Identity.Nonce == r.SourcePark && park.Closed && park.SuccessfulAttempt > 0 && !park.Tombstoned && park.Identity.PID == r.Source.Helper.PID && park.Identity.ProcessIdentity == r.Source.Helper.Identity {
			return true
		}
	}
	return false
}
