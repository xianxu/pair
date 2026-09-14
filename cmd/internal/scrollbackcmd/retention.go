package scrollbackcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func acquireRenderLease(getenv func(string) string, raw, events string) (*storagegc.ProcessLease, error) {
	if getenv == nil {
		return nil, errors.New("renderer requires explicit environment reader")
	}
	dir, tag := getenv("PAIR_DATA_DIR"), getenv("PAIR_TAG")
	if dir == "" && tag == "" {
		return nil, nil
	}
	owner, err := storagegc.SelectedOwner(dir, getenv("PAIR_SCOPE_KEY"), tag)
	if err != nil {
		return nil, err
	}
	raw, err = canonicalSource(raw)
	if err != nil {
		return nil, err
	}
	if events != "" {
		events, err = canonicalSource(events)
		if err != nil {
			return nil, err
		}
	}
	// A captured source carries an independent age and must publish an exact
	// target; a live scrollback source uses the owner-wide writer generation.
	member, matchErr := artifactpath.MatchArtifact(raw, []artifactpath.StorageOwner{owner}, nil)
	target, expectedEvents := "", ""
	if matchErr == nil && member.Family == "parked-scrollback" {
		capture, err := artifactpath.ParseParkedCapture(member, time.Local)
		if err != nil || raw != capture.Raw {
			return nil, errors.New("renderer requires the exact parked raw capture")
		}
		target, expectedEvents = raw, capture.Events
	} else {
		paths, _ := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
		live, err := paths.ScrollbackArtifacts(getenv("PAIR_AGENT"))
		if err != nil || raw != live.Raw {
			return nil, errors.New("renderer source does not belong to selected owner")
		}
		expectedEvents = live.Events
	}
	if events != "" && events != expectedEvents {
		return nil, errors.New("renderer events differ from source capture")
	}
	for _, path := range []string{raw, events} {
		if path == "" {
			continue
		}
		if st, err := os.Lstat(path); err == nil && st.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("renderer source is a symlink")
		}
	}
	return storagegc.AcquireSelectedProcessTarget(context.Background(), getenv, "scrollback-reader", target)
}

func canonicalSource(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.New("managed source must be absolute")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(path)), nil
}

func acknowledgeRenderHandoff(lease *storagegc.ProcessLease, id, raw string) error {
	if id == "" {
		return nil
	}
	if lease == nil {
		return errors.New("handoff requires explicit owner")
	}
	target, err := canonicalSource(raw)
	if err != nil {
		return err
	}
	state, err := lease.Coordinator.ReadOwner(lease.Owner)
	if err != nil {
		return err
	}
	for _, intent := range state.Intents {
		if intent.ID != id {
			continue
		}
		if intent.Target != target {
			return errors.New("handoff target differs from source")
		}
		return lease.Coordinator.CancelUnchangedUse(context.Background(), lease.Owner, id)
	}
	// Re-running a renderer argv after a completed transfer still has a newly
	// acquired exact-target lease and requires no second use publication.
	return nil
}
