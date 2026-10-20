package launcher

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

const maxRecoveryRecordBytes = 2 << 20

// ValidateRecoveryRecord is the pure check of a recovery authority against the
// thread record it names; Couch's tests run it on the authority they launch.
func ValidateRecoveryRecord(a RecoveryAuthority, r threadrecord.Record, scope RepoScope, tag, agent, cwd string, pid int, identity string, now time.Time) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if a.Address.RepoScope != scope.Key || a.Address.Tag != tag || r.Address != a.Address || r.WorkingPath != cwd || r.Revision < a.RecordRevision || identity == "" || pid <= 0 {
		return errors.New("recovery thread address, revision, cwd or helper identity changed")
	}
	if a.Intent.Kind() == threadrecord.RecoveryInferred {
		c := a.Intent.Context
		if c.SourceKind() == sessioninventory.RecoverySourceNative {
			if a.Intent.SourceRevision > a.RecordRevision {
				return errors.New("native recovery source revision changed")
			}
			if err := threadrecord.ValidateNativeRecoverySourceAt(c.Source, now, a.Intent.SourceRevision); err != nil {
				return err
			}
		} else if now.Before(c.CreatedAt) || !now.Before(c.ExpiresAt) || !c.ExpiresAt.Equal(c.Capture.CapturedAt.Add(storagegc.CaptureRetentionPeriod)) {
			return errors.New("recovery evidence expired or retention changed")
		}
	} else if now.Before(a.Intent.CreatedAt) || a.Intent.SourceRevision > a.RecordRevision || a.Intent.Agent != agent {
		return errors.New("explicit recovery source revision, time or agent changed")
	}
	if len(r.Incarnations) != 1 {
		return errors.New("recovery requires its exact creating incarnation")
	}
	incarnation := r.Incarnations[0]
	if incarnation.State != "creating" || incarnation.PID != pid || incarnation.Identity != identity || incarnation.Start == nil || incarnation.Start.Nonce != a.Attempt || !reflect.DeepEqual(incarnation.Start.RecoveryIntent, &a.Intent) || incarnation.Start.LaunchProfile == nil || incarnation.Start.LaunchProfile.Agent != agent {
		return errors.New("recovery start claim or helper changed")
	}
	if a.Intent.Kind() != threadrecord.RecoveryInferred && incarnation.StartedAt.Before(a.Intent.CreatedAt) {
		return errors.New("recovery start predates explicit intent")
	}
	return nil
}

func (r OSRuntime) ValidateRecoveryAuthority(ctx context.Context, a RecoveryAuthority, scope RepoScope, tag, agent, cwd string) error {
	if ctx == nil {
		return errors.New("recovery authority requires context")
	}
	ctx, cancel := context.WithTimeout(ctx, sessioninventory.RecoveryScanTimeout)
	defer cancel()
	if err := a.Validate(); err != nil {
		return err
	}
	// The private profile supplies the locator for either global or slot-local
	// storage. Its contents and the executing helper are the authority, not the
	// pathname alone. Do not follow a changed record directory alias.
	physical, err := filepath.EvalSymlinks(filepath.Dir(a.RecordPath))
	if err != nil {
		return err
	}
	if physical != filepath.Dir(a.RecordPath) {
		return errors.New("recovery record directory changed")
	}
	raw, err := osfs.ReadRegularMetadata(ctx, a.RecordPath, maxRecoveryRecordBytes)
	if err != nil {
		return err
	}
	record, err := threadrecord.DecodePersisted(raw, a.Address, threadrecord.Validators{RepoScope: ValidateRepoScopeKey, Tag: ValidatePairTag})
	if err != nil {
		return err
	}
	actualCWD, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return err
	}
	recordCWD, err := filepath.EvalSymlinks(record.WorkingPath)
	if err != nil {
		return err
	}
	if actualCWD != recordCWD {
		return errors.New("recovery physical cwd changed")
	}
	if err = ValidateRecoveryRecord(a, record, scope, tag, agent, cwd, os.Getpid(), procutil.Identity(strconv.Itoa(os.Getpid())), time.Now()); err != nil {
		return err
	}
	// Tagged orientation is descriptive data. Losing an optional screen cannot
	// revoke a proven, Fresh or native-only selection. Capture-backed inference
	// retains its exact archive requirements.
	if a.Intent.Kind() == threadrecord.RecoveryInferred {
		var checkCapture func() error
		if a.Intent.Context.SourceKind() == sessioninventory.RecoverySourceCapture {
			checkCapture = func() error {
				if err := ctx.Err(); err != nil {
					return err
				}
				c := a.Intent.Context.Capture
				rawID, err := storagegc.CaptureIdentity(c.Raw)
				if err != nil {
					return err
				}
				eventsID, err := storagegc.CaptureIdentity(c.Events)
				if err != nil {
					return err
				}
				if rawID != c.RawIdentity || eventsID != c.EventsIdentity {
					return errors.New("recovery capture generation changed")
				}
				return nil
			}
			if err = checkCapture(); err != nil {
				return err
			}
			c := a.Intent.Context.Capture
			paths, err := artifactpath.ResolveScoped(r.DataDir, tag)
			if err != nil {
				return err
			}
			archived, err := paths.ParkedScrollbackArtifacts(c.Token)
			if err != nil || c.Token == "" || archived.Raw != c.Raw || archived.Events != c.Events {
				return errors.New("recovery capture is not the exact owner's archived screen")
			}
			metadata, err := osfs.ReadRegularMetadata(ctx, archived.Metadata, 8192)
			if err != nil {
				return err
			}
			member := artifactpath.ParkedCapture{Raw: archived.Raw, Events: archived.Events, Metadata: archived.Metadata}
			captureMeta, err := artifactpath.DecodeCaptureMetadata(metadata, member)
			if err != nil {
				return err
			}
			if captureMeta.RawIdentity != c.RawIdentity || captureMeta.EventsIdentity == nil || *captureMeta.EventsIdentity != c.EventsIdentity || !captureMeta.CapturedAt.Equal(c.CapturedAt) {
				return errors.New("recovery capture metadata changed")
			}
			if c.LaunchOrdinal != 0 {
				events, err := osfs.ReadRegularMetadata(ctx, c.Events, artifactpath.MaxCaptureWitnessEventsBytes)
				if err != nil {
					return err
				}
				witness, err := artifactpath.DecodeCaptureWitness(events)
				if err != nil {
					return err
				}
				if witness.LaunchOrdinal != c.LaunchOrdinal || witness.Nonce != c.Nonce || witness.ScopeKey != a.Address.RepoScope || witness.Tag != tag {
					return errors.New("recovery capture producer changed")
				}
			}
		}
		if a.Intent.NativeID != "" {
			var selected *threadrecord.RecoveryCandidate
			for i := range a.Intent.Context.Candidates {
				candidate := &a.Intent.Context.Candidates[i]
				if candidate.Key == a.Intent.SelectedKey {
					selected = candidate
					break
				}
			}
			if selected == nil || selected.Agent != agent {
				return errors.New("recovery native selection changed")
			}
			entry := sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: selected.ArtifactRoot, RelativePath: selected.ArtifactRelativePath}, StableFileID: sessioninventory.StableFileID(selected.StableFileID), GenerationToken: sessioninventory.GenerationToken(selected.GenerationToken), MutationToken: sessioninventory.MutationToken(selected.MutationToken), Size: selected.Size, ModTime: &selected.ModTime}
			c := a.Intent.Context
			var cutoff time.Time
			if c.SourceKind() == sessioninventory.RecoverySourceNative {
				if selected.CWD != actualCWD {
					return errors.New("native recovery candidate physical cwd changed")
				}
				cutoff = c.Source.Cutoff
			} else {
				cutoff = c.Capture.LastActivity
				if cutoff.IsZero() {
					cutoff = c.Capture.RawIdentity.ModifiedAt
				}
			}
			native, err := r.nativeInventory(agent, "")
			if err != nil {
				return err
			}
			if err = sessioninventory.RevalidateRecoveryCandidate(ctx, native, entry, sessioninventory.Agent(agent), a.Intent.NativeID, cwd, cutoff); err != nil {
				return err
			}
		}
		if checkCapture != nil {
			if err = checkCapture(); err != nil {
				return err
			}
		}
	} else if a.Intent.Kind() == threadrecord.RecoveryProven {
		runtime, err := r.nativeInventory(agent, "")
		if err != nil {
			return err
		}
		target, err := sessioninventory.QueryResumeTargetContext(ctx, runtime, scope.Key, tag, sessioninventory.Agent(agent))
		if err != nil {
			return err
		}
		if err := RequireNativeResumeBinding(a.Intent.NativeID, target.NativeID, target.Status); err != nil {
			return err
		}
	}
	after, err := osfs.ReadRegularMetadata(ctx, a.RecordPath, maxRecoveryRecordBytes)
	if err != nil {
		return err
	}
	if !bytes.Equal(raw, after) {
		return errors.New("recovery start claim changed during validation")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return ValidateRecoveryRecord(a, record, scope, tag, agent, cwd, os.Getpid(), procutil.Identity(strconv.Itoa(os.Getpid())), time.Now())
}
