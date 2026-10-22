package remotesession

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/durablefile"
)

// A sole leased Couch writer owns one bounded reusable stage per record. Stages
// are never committed authority. Reopening validates the committed records first,
// then reclaims only their exact regular bounded slots; no glob/recurse/repair.
func reclaimRemoteStage(path string, limit int) error {
	stage, err := artifactpath.RemotePublicationStage(path)
	if err != nil {
		return err
	}
	info, err := os.Lstat(stage)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > int64(limit) {
		return errors.New("remote publication stage unsafe; inspect locally")
	}
	if err := os.Remove(stage); err != nil {
		return err
	}
	return durablefile.SyncDirectory(filepath.Dir(stage))
}
func writeRemoteAtomic(path string, raw []byte, limit int) error {
	if len(raw) > limit {
		return errors.New("remote publication exceeds bound")
	}
	if err := reclaimRemoteStage(path, limit); err != nil {
		return err
	}
	stage, err := artifactpath.RemotePublicationStage(path)
	if err != nil {
		return err
	}
	return durablefile.WriteAtomicStaged(path, raw, stage)
}
