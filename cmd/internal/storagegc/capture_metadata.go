package storagegc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

func CaptureIdentity(path string) (artifactpath.CaptureFileIdentity, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return artifactpath.CaptureFileIdentity{}, err
	}
	native, ok := st.Sys().(*syscall.Stat_t)
	if !st.Mode().IsRegular() || !ok {
		return artifactpath.CaptureFileIdentity{}, errors.New("unsafe capture member")
	}
	return artifactpath.CaptureFileIdentity{Size: st.Size(), ModifiedAt: st.ModTime().UTC(), Device: uint64(native.Dev), Inode: uint64(native.Ino)}, nil
}

// PublishCaptureMetadata follows completed payload syncs. The caller keeps its
// producer registration until publication returns; interrupted publication leaves
// the payload recoverable under the conservative legacy clock.
func (c *Coordinator) PublishCaptureMetadata(ctx context.Context, owner artifactpath.StorageOwner, capture artifactpath.ParkedCapture, at time.Time, witnesses ...*artifactpath.CaptureWitness) error {
	if err := c.validateOwner(owner); err != nil {
		return err
	}
	member, err := artifactpath.MatchArtifact(capture.Raw, []artifactpath.StorageOwner{owner}, nil)
	if err != nil {
		return err
	}
	parsed, err := artifactpath.ParseParkedCapture(member, time.UTC)
	if err != nil || parsed.Raw != capture.Raw || parsed.Events != capture.Events || parsed.Metadata != capture.Metadata || at.IsZero() {
		return errors.New("invalid capture publication")
	}
	var witness *artifactpath.CaptureWitness
	if len(witnesses) > 1 {
		return errors.New("multiple capture witnesses")
	}
	if len(witnesses) == 1 {
		witness = witnesses[0]
	}
	if witness != nil {
		copy := *witness
		witness = &copy
		if err := artifactpath.ValidateCaptureWitness(*witness); err != nil {
			return err
		}
		if witness.ScopeKey != owner.RepoScope || witness.Tag != owner.Tag {
			return errors.New("foreign capture witness")
		}
	}
	return c.WithLock(ctx, func(l *Locked) error {
		if _, err := os.Lstat(capture.Metadata); !errors.Is(err, os.ErrNotExist) {
			return errors.New("capture metadata already exists or is unavailable")
		}
		raw, err := CaptureIdentity(capture.Raw)
		if err != nil {
			return err
		}
		var events *artifactpath.CaptureFileIdentity
		e, err := CaptureIdentity(capture.Events)
		if err == nil {
			events = &e
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		version := 1
		if witness != nil {
			archived, err := readCaptureWitnessEvents(capture.Events)
			if err != nil {
				return err
			}
			if archived == nil || *archived != *witness {
				return errors.New("archived capture witness changed")
			}
			version = 2
		}
		return l.atomicJSON(capture.Metadata, artifactpath.CaptureMetadata{Version: version, Witness: witness, Raw: filepath.Base(capture.Raw), Events: filepath.Base(capture.Events), CapturedAt: at.UTC(), RawIdentity: raw, EventsIdentity: events})
	})
}

// capturePublicationTime trusts the producer clock only while the completed
// payload identities still match. Absent metadata uses the legacy fallback.
func capturePublicationTime(capture artifactpath.ParkedCapture) (time.Time, bool, error) {
	var raw json.RawMessage
	if err := readStateJSON(capture.Metadata, &raw); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return time.Time{}, false, nil
		}
		return time.Time{}, true, err
	}
	metadata, err := artifactpath.DecodeCaptureMetadata(raw, capture)
	if err != nil {
		return time.Time{}, true, err
	}
	same := func(path string, want artifactpath.CaptureFileIdentity) bool {
		got, err := CaptureIdentity(path)
		return err == nil && got.Size == want.Size && got.ModifiedAt.Equal(want.ModifiedAt) && got.Device == want.Device && got.Inode == want.Inode
	}
	if !same(capture.Raw, metadata.RawIdentity) {
		return time.Time{}, true, errors.New("capture raw identity changed")
	}
	if metadata.EventsIdentity != nil {
		if !same(capture.Events, *metadata.EventsIdentity) {
			return time.Time{}, true, errors.New("capture events identity changed")
		}
	} else if _, err := os.Lstat(capture.Events); !errors.Is(err, os.ErrNotExist) {
		return time.Time{}, true, errors.New("capture events appeared after publication")
	}
	return metadata.CapturedAt, true, nil
}

// ReadCaptureWitness verifies the currently open generation's source file IDs.
// Missing historical authority is explicit nil; malformed authority is an error.
func ReadCaptureWitness(rawPath, eventsPath string) (*artifactpath.CaptureWitness, error) {
	w, e := readCaptureWitnessEvents(eventsPath)
	if e != nil || w == nil {
		return w, e
	}
	raw, e := CaptureIdentity(rawPath)
	if e != nil {
		return nil, e
	}
	events, e := CaptureIdentity(eventsPath)
	if e != nil {
		return nil, e
	}
	if w.RawFile != (artifactpath.CaptureFileID{Device: raw.Device, Inode: raw.Inode}) || w.EventsFile != (artifactpath.CaptureFileID{Device: events.Device, Inode: events.Inode}) {
		return nil, errors.New("capture witness file generation changed")
	}
	return w, nil
}
func readCaptureWitnessEvents(path string) (*artifactpath.CaptureWitness, error) {
	f, e := os.Open(path)
	if errors.Is(e, os.ErrNotExist) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	defer f.Close()
	data, e := io.ReadAll(io.LimitReader(f, artifactpath.MaxCaptureWitnessEventsBytes+1))
	if e != nil {
		return nil, e
	}
	w, e := artifactpath.DecodeCaptureWitness(data)
	if errors.Is(e, artifactpath.ErrCaptureWitnessAbsent) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return &w, nil
}
