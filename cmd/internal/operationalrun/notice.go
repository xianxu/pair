package operationalrun

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/durablefile"
	"golang.org/x/sys/unix"
)

// FinalizeNotice monotonically finalizes delivery for a known completed run.
// Execution facts, additive metadata and the output file remain unchanged.
func (s Store) FinalizeNotice(ctx context.Context, key RunKey, revision, notice string) error {
	if !terminalNotice(notice) {
		return errors.New("invalid terminal operational notice")
	}
	task, err := s.task(key)
	if err != nil {
		return err
	}
	unlock, err := taskLock(ctx, task, false)
	if err != nil {
		return err
	}
	defer unlock()
	if err = s.cleanupStages(ctx, task); err != nil {
		return err
	}
	r, err := s.readMetadata(key)
	if err != nil {
		return err
	}
	if r.DeclarationRevision != revision || !r.Completed() {
		return errors.New("notice finalization requires matching completed execution")
	}
	run := filepath.Join(task, "runs", key.ID)
	if r.Notice == notice {
		return errors.Join(durablefile.SyncDirectory(run), durablefile.SyncDirectory(task))
	}
	if r.Notice != "pending" {
		return errors.New("operational notice is not pending")
	}
	r.Notice = notice
	raw, err := encodeRecord(r)
	if err != nil {
		return err
	}
	fd, err := unix.Open(task, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	root := os.NewFile(uintptr(fd), task)
	defer root.Close()
	rootID, err := stageOpenedIdentity(root)
	if err != nil {
		return err
	}
	path, err := os.MkdirTemp(task, ".run-stage-")
	if err != nil {
		return err
	}
	name := filepath.Base(path)
	stage, err := stageOpenChild(root, name, true)
	if err != nil {
		return err
	}
	defer stage.Close()
	id, err := stageOpenedIdentity(stage)
	if err != nil {
		return err
	}
	files := map[string]fileIdentity{}
	var children []string
	defer func() {
		_ = (Store{}).cleanupCapturedRunStage(context.Background(), root, rootID, name, stage, id, children, files)
	}()
	if err = durablefile.WriteAtomic(filepath.Join(path, "record.json"), raw, ".record-"); err != nil {
		return err
	}
	files["record.json"], err = stageChildIdentity(stage, "record.json", false)
	if err != nil {
		return err
	}
	children = []string{"record.json"}
	if err = s.fail("notice-ready"); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = verifyCapturedRunStage(root, rootID, name, stage, id, files); err != nil {
		return err
	}
	if err = os.Rename(filepath.Join(path, "record.json"), filepath.Join(run, "record.json")); err != nil {
		return err
	}
	delete(files, "record.json")
	children = nil
	if err = s.fail("notice-renamed"); err != nil {
		return err
	}
	if err = (Store{}).cleanupCapturedRunStage(ctx, root, rootID, name, stage, id, children, files); err != nil {
		return err
	}
	return errors.Join(durablefile.SyncDirectory(run), durablefile.SyncDirectory(task))
}

func terminalNotice(notice string) bool {
	switch notice {
	case "submitted", "expired", "uncertain":
		return true
	}
	if len(notice) > 4096 || !utf8.ValidString(notice) || !strings.HasPrefix(notice, "dropped: ") {
		return false
	}
	reason := strings.TrimPrefix(notice, "dropped: ")
	if strings.TrimSpace(reason) == "" {
		return false
	}
	return !strings.ContainsFunc(reason, unicode.IsControl)
}
