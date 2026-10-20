package historyarchive

import (
	"context"
	"errors"
	"path"
	"strings"
	"time"
	"unicode/utf8"
)

// ResolveChild binds a single child using only the caller's current flat
// observation. It never resolves an ambiguous name by selecting one match.
func (r Rclone) ResolveChild(ctx context.Context, parent PinnedDrive, name string, inventory map[string][]RemoteEntry) (PinnedDrive, bool, error) {
	if err := ctx.Err(); err != nil {
		return PinnedDrive{}, false, err
	}
	if !flatName(name) || !utf8.ValidString(name) || strings.ContainsRune(name, ':') {
		return PinnedDrive{}, false, errors.New("invalid Drive child folder name")
	}
	if err := r.validateFolderParent(parent); err != nil {
		return PinnedDrive{}, false, err
	}
	matches := inventory[name]
	if len(matches) == 0 {
		return PinnedDrive{}, false, nil
	}
	if len(matches) != 1 || matches[0].Name != name || !matches[0].IsDir {
		return PinnedDrive{}, false, errors.New("Drive child folder conflicts with observed entries")
	}
	child := PinnedDrive{Remote: parent.Remote, Root: parent.Remote + ":", FolderID: matches[0].ID}
	if r.LocalConformance {
		child.Root = parent.Root + "/" + name
		if child.FolderID == "" {
			child.FolderID = child.Root
		}
	}
	if !validRemoteIdentity(child.FolderID) {
		return PinnedDrive{}, false, errors.New("Drive child folder lacks stable ID")
	}
	return child, true, nil
}

func (r Rclone) validateFolderParent(parent PinnedDrive) error {
	remote, err := ParseDriveDestination(parent.Remote, "parent")
	if err != nil || parent.Remote == "" || remote.Remote != parent.Remote {
		return errors.New("invalid Drive parent remote")
	}
	if !r.LocalConformance {
		if parent.Root != parent.Remote+":" || !validRemoteIdentity(parent.FolderID) {
			return errors.New("invalid pinned Drive parent")
		}
		return nil
	}
	prefix := parent.Remote + ":"
	if !strings.HasPrefix(parent.Root, prefix) || !validRemoteIdentity(parent.Root) || parent.FolderID != "" && !validRemoteIdentity(parent.FolderID) {
		return errors.New("invalid local Drive parent")
	}
	folder := strings.TrimPrefix(parent.Root, prefix)
	if folder == "" || folder == "." || folder == ".." || strings.HasPrefix(folder, "../") || strings.ContainsAny(folder, "\\:") || path.Clean(folder) != folder {
		return errors.New("invalid local Drive parent path")
	}
	return nil
}

// EnsureChild performs at most one create attempt, followed by one confirming
// observation of the same pinned parent. A lost mkdir acknowledgment can be
// reconciled by positive evidence; an absent or failed observation cannot.
func (r Rclone) EnsureChild(ctx context.Context, parent PinnedDrive, name string, inventory map[string][]RemoteEntry) (PinnedDrive, error) {
	if err := ctx.Err(); err != nil {
		return PinnedDrive{}, err
	}
	state, _, err := StepDriveFolder(DriveFolderState{}, DriveFolderEvent{Kind: FolderBegin})
	if err != nil {
		return PinnedDrive{}, err
	}
	observe := func(entries map[string][]RemoteEntry) (PinnedDrive, error) {
		child, found, resolveErr := r.ResolveChild(ctx, parent, name, entries)
		event := DriveFolderEvent{Kind: FolderAbsent}
		switch {
		case ctx.Err() != nil:
			event.Kind = FolderCancel
			resolveErr = errors.Join(resolveErr, ctx.Err())
		case resolveErr != nil:
			event.Kind = FolderAmbiguous
		case found:
			event = DriveFolderEvent{Kind: FolderUnique, ID: child.FolderID}
		}
		var stepErr error
		state, _, stepErr = StepDriveFolder(state, event)
		return child, errors.Join(resolveErr, stepErr)
	}
	child, err := observe(inventory)
	if err != nil {
		return PinnedDrive{}, err
	}
	if state.Phase() == FolderReady {
		return child, nil
	}
	var effect DriveFolderEffect
	state, effect, err = StepDriveFolder(state, DriveFolderEvent{Kind: FolderCreate})
	if err != nil || effect != FolderMake {
		return PinnedDrive{}, errors.Join(err, errors.New("Drive child create not admitted"))
	}
	if err := ctx.Err(); err != nil {
		return PinnedDrive{}, err
	}
	target := parent.Root + name
	if r.LocalConformance {
		target = parent.Root + "/" + name
	}
	bounded, cancel := context.WithTimeout(ctx, 60*time.Second)
	_, mkdirErr := r.runner().Run(bounded, r.pinnedArgs(parent, "mkdir", target), 16<<10)
	cancel()
	if err := ctx.Err(); err != nil {
		_, _, _ = StepDriveFolder(state, DriveFolderEvent{Kind: FolderCancel})
		return PinnedDrive{}, errors.Join(mkdirErr, err)
	}
	// An error can describe a late remote effect. Re-observe instead of issuing
	// another mkdir or treating the acknowledgment as identity evidence.
	state, effect, err = StepDriveFolder(state, DriveFolderEvent{Kind: FolderCreated})
	if err != nil || effect != FolderObserve {
		return PinnedDrive{}, errors.Join(mkdirErr, err, errors.New("Drive child confirmation not admitted"))
	}
	after, observeErr := r.ObserveFlatInventory(ctx, parent)
	if err := ctx.Err(); err != nil {
		_, _, _ = StepDriveFolder(state, DriveFolderEvent{Kind: FolderCancel})
		return PinnedDrive{}, errors.Join(mkdirErr, observeErr, err)
	}
	if observeErr != nil {
		_, _, _ = StepDriveFolder(state, DriveFolderEvent{Kind: FolderFailure})
		return PinnedDrive{}, errors.Join(mkdirErr, observeErr)
	}
	child, err = observe(after)
	if err != nil {
		return PinnedDrive{}, errors.Join(mkdirErr, err)
	}
	if state.Phase() != FolderReady {
		return PinnedDrive{}, errors.Join(mkdirErr, errors.New("Drive child folder creation remains unconfirmed"))
	}
	return child, nil
}
