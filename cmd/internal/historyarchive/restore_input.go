package historyarchive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type stagedRestoreInput struct {
	Plan RestoreBundlePlan
	File *os.File
}

// Acquisition never uses index evidence as a substitute for hashing the actual
// private ciphertext. Every selected input is verified before any is decoded.
func (r Restorer) acquireRestoreInputs(ctx context.Context, plan RestorePlan, stage *OwnedStaging, dir *os.File, budget *RestoreExtractionBudget) (inputs []stagedRestoreInput, err error) {
	defer func() {
		if err != nil {
			for _, input := range inputs {
				input.File.Close()
			}
		}
	}()
	var local *os.File
	if plan.Request.From != "drive" {
		local, err = openSafeDir(r.Config.Sink, false)
		if err != nil && !errors.Is(err, unix.ENOENT) {
			return nil, err
		}
		if local != nil {
			defer local.Close()
		}
	}
	var remote []RestoreBundlePlan
	for _, bundle := range plan.Bundles {
		if err = ctx.Err(); err != nil {
			return inputs, err
		}
		var source *os.File
		if local != nil {
			source, err = openRegularAt(local, bundle.BundleKey, unix.O_RDONLY)
			if err != nil && !errors.Is(err, unix.ENOENT) {
				return inputs, err
			}
		}
		if source == nil {
			if plan.Request.From == "local" {
				return inputs, fmt.Errorf("local bundle %s is absent", bundle.Label.ID)
			}
			remote = append(remote, bundle)
			continue
		}
		out, createErr := CreateRestoreFile(dir, bundle.Label.ID+".tar.zst.gpg", budget)
		if createErr != nil {
			source.Close()
			return inputs, createErr
		}
		n, hash, copyErr := fileDigest(ctx, source, out, bundle.Label.Bytes)
		copyErr = errors.Join(copyErr, source.Close(), out.Sync(), out.Close())
		if copyErr != nil {
			return inputs, copyErr
		}
		if n != bundle.Label.Bytes || hash != bundle.Label.SHA256 {
			return inputs, fmt.Errorf("local bundle %s ciphertext checksum/size conflict; inspect it or explicitly select Drive", bundle.Label.ID)
		}
	}
	if len(remote) > 0 {
		dest, parseErr := ParseDriveDestination(r.Config.DriveRemote, r.Config.DriveFolder)
		if parseErr != nil {
			return inputs, parseErr
		}
		pin, resolveErr := r.Rclone.ResolveDestination(ctx, dest)
		if resolveErr != nil {
			return inputs, resolveErr
		}
		inventory, observeErr := r.Rclone.ObserveFlatInventory(ctx, pin)
		if observeErr != nil {
			return inputs, observeErr
		}
		names := make([]string, 0, len(remote))
		var remoteBytes, maxRemoteBytes int64
		for _, bundle := range remote {
			name := bundle.Label.ID + ".tar.zst.gpg"
			entries := inventory[name]
			if len(entries) != 1 || entries[0].IsDir || entries[0].Size != bundle.Label.Bytes || (!r.Rclone.LocalConformance && !validRemoteIdentity(entries[0].ID)) {
				return inputs, fmt.Errorf("Drive bundle %s is absent, ambiguous or has conflicting size/identity", bundle.Label.ID)
			}
			names = append(names, name)
			remoteBytes += bundle.Label.Bytes
			maxRemoteBytes = max(maxRemoteBytes, bundle.Label.Bytes)
		}
		// Reserve the batch's files, filter and up to eight concurrent transfer
		// temporaries before giving an external process filesystem access.
		reserved := len(names) + 9
		if budget.Entries > MaxRestoreEntries-reserved {
			return inputs, errors.New("restore acquisition entry budget exhausted")
		}
		budget.Entries += reserved
		stagePath := filepath.Join(filepath.Dir(plan.Request.To), ".pair-restore-staging")
		if err = r.Rclone.DownloadBatch(ctx, pin, names, filepath.Join(stagePath, "ciphertext"), filepath.Join(stagePath, "files-from"), remoteBytes, maxRemoteBytes); err != nil {
			return inputs, err
		}
	}
	for _, bundle := range plan.Bundles {
		file, openErr := openRegularAt(dir, bundle.Label.ID+".tar.zst.gpg", unix.O_RDONLY)
		if openErr != nil {
			return inputs, openErr
		}
		inputs = append(inputs, stagedRestoreInput{Plan: bundle, File: file})
		n, hash, digestErr := fileDigest(ctx, file, nil, bundle.Label.Bytes)
		if digestErr != nil {
			return inputs, digestErr
		}
		if n != bundle.Label.Bytes || hash != bundle.Label.SHA256 {
			return inputs, fmt.Errorf("bundle %s ciphertext checksum/size conflict", bundle.Label.ID)
		}
		if _, err = file.Seek(0, io.SeekStart); err != nil {
			return inputs, err
		}
	}
	return inputs, nil
}
