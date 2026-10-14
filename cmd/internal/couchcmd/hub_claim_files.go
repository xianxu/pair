package couchcmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/durablefile"
	"golang.org/x/sys/unix"
)

// Hub configuration is read only for explicitly started hubs. Missing is empty;
// unreadable, oversized and malformed files are never treated as missing.
func ReadHubClaimFile(root string) (couchcore.HubClaimDocument, error) {
	raw, _, err := readHubClaimBytes(root)
	if err != nil {
		return couchcore.HubClaimDocument{}, err
	}
	doc, err := couchcore.ParseHubClaims(raw)
	if err != nil {
		return doc, fmt.Errorf("%s: %w", filepath.Join(root, ".hub"), err)
	}
	return doc, nil
}
func readHubClaimBytes(root string) ([]byte, os.FileMode, error) {
	physical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, 0, err
	}
	if physical != root || !filepath.IsAbs(root) {
		return nil, 0, fmt.Errorf("hub directory identity changed: %s", root)
	}
	dir, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, 0, err
	}
	defer unix.Close(dir)
	path := filepath.Join(root, ".hub")
	fd, err := unix.Openat(dir, ".hub", unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if os.IsNotExist(err) {
		return nil, 0600, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", path, err)
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	if !info.Mode().IsRegular() || info.Size() > couchcore.MaxHubClaimBytes {
		return nil, 0, fmt.Errorf("%s: expected a regular file at most 64 KiB; shorten or repair it and retry", path)
	}
	raw, err := io.ReadAll(io.LimitReader(f, couchcore.MaxHubClaimBytes+1))
	if err != nil {
		return nil, 0, err
	}
	if len(raw) > couchcore.MaxHubClaimBytes {
		return nil, 0, fmt.Errorf("%s: file grew past 64 KiB; shorten it and retry", path)
	}
	return raw, info.Mode().Perm(), nil
}

// One console operation serializes these writes. The seam models the real
// filesystem across calls, including a failure after rename; it is not a claim
// that several independent hub files can be atomically committed together.
type hubClaimFiles struct {
	write         func(string, []byte, os.FileMode) error
	syncDirectory func(string) error
}

func (f hubClaimFiles) Apply(ctx context.Context, edits []couchcore.HubClaimEdit) error {
	if len(edits) > couchcore.MaxHubSlots {
		return fmt.Errorf("too many hub claim edits")
	}
	check := func(e couchcore.HubClaimEdit) (bool, os.FileMode, error) {
		if err := ctx.Err(); err != nil {
			return false, 0, err
		}
		if _, err := couchcore.ParseHubClaims(e.After); err != nil {
			return false, 0, err
		}
		raw, mode, err := readHubClaimBytes(e.Root)
		if err != nil {
			return false, 0, err
		}
		if bytes.Equal(raw, e.After) {
			return true, mode, nil
		}
		if !bytes.Equal(raw, e.Before) {
			return false, 0, fmt.Errorf("%s changed; inspect the file and retry", filepath.Join(e.Root, ".hub"))
		}
		return false, mode, nil
	}
	for _, e := range edits {
		if _, _, err := check(e); err != nil {
			return err
		}
	}
	write := f.write
	if write == nil {
		write = func(path string, raw []byte, mode os.FileMode) error {
			return durablefile.WriteAtomicMode(path, raw, ".hub-", mode)
		}
	}
	syncDirectory := f.syncDirectory
	if syncDirectory == nil {
		syncDirectory = durablefile.SyncDirectory
	}
	for _, e := range edits {
		done, mode, err := check(e)
		if err == nil {
			if done {
				err = syncDirectory(e.Root)
			} else {
				err = write(filepath.Join(e.Root, ".hub"), e.After, mode)
			}
		}
		if err != nil {
			return fmt.Errorf("hub persistence incomplete at %s: %w; inspect claims and retry the move (earlier files may already be saved)", filepath.Join(e.Root, ".hub"), err)
		}
	}
	return nil
}
