package couchcore

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// ReadRetainedTerminal treats tagged orientation as display-only coordinates.
// It reconstructs their exact owner under this reader's configured scope/root;
// persisted paths never become arbitrary renderer inputs or native authority.
func (r OSRecoveryEvidenceReader) ReadRetainedTerminal(ctx context.Context, address ThreadAddress, agent string, capture threadrecord.RecoveryCapture, maxLines int) (TerminalRecording, error) {
	if ctx == nil {
		return TerminalRecording{}, errors.New("retained terminal requires context")
	}
	if err := ctx.Err(); err != nil {
		return TerminalRecording{}, err
	}
	root, err := filepath.EvalSymlinks(r.DataDir)
	if err != nil {
		return TerminalRecording{}, err
	}
	if _, err := artifactpath.Resolve(artifactpath.Address{DataDir: root, RepoScope: address.RepoScope, Tag: string(address.Tag)}); err != nil {
		return TerminalRecording{}, err
	}
	if capture.Token == "" {
		return TerminalRecording{}, errors.New("retained terminal token unavailable")
	}
	owners, err := artifactpath.DiscoverStorageOwners(root, address.RepoScope, []string{filepath.Base(capture.Raw)})
	if err != nil {
		return TerminalRecording{}, err
	}
	if len(owners) != 1 {
		return TerminalRecording{}, errors.New("retained terminal owner unavailable")
	}
	member, err := artifactpath.MatchArtifact(capture.Raw, owners, nil)
	if err != nil {
		return TerminalRecording{}, err
	}
	archived, err := artifactpath.ParseParkedCapture(member, time.UTC)
	if err != nil || archived.Raw != capture.Raw || archived.Events != capture.Events || archived.Token != capture.Token {
		return TerminalRecording{}, errors.New("retained terminal members do not match exact owner/token")
	}
	paths, err := artifactpath.ResolveScoped(owners[0].Directory(), owners[0].Tag)
	if err != nil {
		return TerminalRecording{}, err
	}
	lease, err := retainedPeekLease(ctx, paths, capture, agent)
	if err != nil {
		return TerminalRecording{}, err
	}
	if lease == nil {
		return TerminalRecording{}, errors.New("retained terminal lease unavailable")
	}
	defer lease.Close()
	metadata, err := recoveryCaptureMetadata(ctx, paths, &capture, agent, nil)
	if err != nil {
		return TerminalRecording{}, err
	}
	if metadata == nil {
		return TerminalRecording{}, errors.New("retained terminal archive metadata unavailable")
	}
	result := TerminalRecording{Raw: archived.Raw, Events: archived.Events}
	result.Lines, err = scrollbackcmd.RenderBoundedLinesContext(ctx, archived.Raw, archived.Events, maxLines)
	if err != nil {
		return TerminalRecording{}, err
	}
	if err := validateRecoveryCaptureSnapshot(ctx, capture); err != nil {
		return TerminalRecording{}, err
	}
	return result, nil
}

var retainedPeekSequence atomic.Uint64

func retainedPeekLease(ctx context.Context, paths artifactpath.Paths, capture threadrecord.RecoveryCapture, agent string) (*storagegc.ProcessLease, error) {
	// Process roles are deliberately reusable across exec, not reference counted.
	// Give each concurrent peek its own registration so releasing one cannot
	// remove another peek's or an inference reader's retention protection.
	role := "retained-peek-" + strconv.FormatUint(retainedPeekSequence.Add(1), 10)
	env := map[string]string{"PAIR_DATA_DIR": paths.ScopeDir(), "PAIR_TAG": paths.Tag(), "PAIR_AGENT": agent}
	return storagegc.AcquireSelectedProcessTarget(ctx, func(key string) string { return env[key] }, role, capture.Raw)
}
