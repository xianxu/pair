package historyarchive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
		if err = r.acquireDriveRestoreInputs(ctx, plan, remote, budget); err != nil {
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

// Validate all recorded candidates before contacting Drive. A selected source is
// binding: stale identity or corrupt bytes never trigger another-copy fallback.
func (r Restorer) acquireDriveRestoreInputs(ctx context.Context, plan RestorePlan, remote []RestoreBundlePlan, budget *RestoreExtractionBudget) error {
	root, err := ParseDriveDestination(r.Config.DriveRemote, r.Config.DriveFolder)
	if err != nil {
		return err
	}
	layout := DriveArchiveLayout{Root: root}
	type group struct {
		location        BundleLocation
		bundles         []RestoreBundlePlan
		bytes, maxBytes int64
		pin             PinnedDrive
	}
	groups := map[string]*group{}
	keys := []string{}
	var total int64
	reserved := 0
	for _, bundle := range remote {
		if len(bundle.DriveLocations) == 0 {
			return fmt.Errorf("Drive bundle %s has no recorded location; run sync-index first", bundle.Label.ID)
		}
		for _, loc := range bundle.DriveLocations {
			if _, err := layout.ValidateRecordedDriveLocation(bundle.Label, loc.Locator); err != nil {
				return err
			}
			if err := validateLocation(loc, bundle.Label); err != nil {
				return err
			}
			if !validRemoteIdentity(loc.FolderID) {
				return errors.New("invalid recorded Drive folder identity")
			}
		}
		loc := bundle.DriveLocations[0]
		key := loc.Identity()
		g := groups[key]
		if g == nil {
			g = &group{location: loc}
			groups[key] = g
			keys = append(keys, key)
			reserved += 9
		}
		if bundle.Label.Bytes > MaxRestoreCipherBytes-total {
			return errors.New("restore acquisition ciphertext budget exhausted")
		}
		total += bundle.Label.Bytes
		reserved++
		g.bundles = append(g.bundles, bundle)
		g.bytes += bundle.Label.Bytes
		g.maxBytes = max(g.maxBytes, bundle.Label.Bytes)
	}
	if budget.Entries > MaxRestoreEntries-reserved {
		return errors.New("restore acquisition entry budget exhausted")
	}
	budget.Entries += reserved
	sort.Strings(keys)
	pin, err := r.Rclone.ResolveDestination(ctx, root)
	if err != nil {
		return err
	}
	// Local backends have no stable directory ID; discovery records the
	// physical path as its conformance-only identity.
	if r.Rclone.LocalConformance && pin.FolderID == "" {
		pin.FolderID = pin.Root
	}
	// One flat observation per visited physical folder, shared by all selections.
	inventories := map[string]map[string][]RemoteEntry{}
	observe := func(p PinnedDrive) (map[string][]RemoteEntry, error) {
		if inventory, ok := inventories[p.FolderID]; ok {
			return inventory, nil
		}
		inventory, err := r.Rclone.ObserveFlatInventory(ctx, p)
		if err == nil {
			inventories[p.FolderID] = inventory
		}
		return inventory, err
	}
	rootInventory, err := observe(pin)
	if err != nil {
		return err
	}
	for _, key := range keys {
		g := groups[key]
		g.pin = pin
		if g.location.Locator != root.String() {
			relative := strings.TrimPrefix(g.location.Locator, root.String()+"/")
			inventory := rootInventory
			parts := strings.Split(relative, "/")
			for n, name := range parts {
				child, found, err := r.Rclone.ResolveChild(ctx, g.pin, name, inventory)
				if err != nil {
					return err
				}
				if !found {
					return errors.New("recorded Drive folder is absent; run sync-index")
				}
				g.pin = child
				if n < len(parts)-1 {
					inventory, err = observe(child)
					if err != nil {
						return err
					}
				}
			}
		}
		if g.pin.FolderID != g.location.FolderID {
			return errors.New("recorded Drive folder identity changed; run sync-index")
		}
		inventory, err := observe(g.pin)
		if err != nil {
			return err
		}
		for _, bundle := range g.bundles {
			entries := inventory[bundle.Label.ID+".tar.zst.gpg"]
			if len(entries) != 1 || entries[0].IsDir || entries[0].Size != bundle.Label.Bytes || (!r.Rclone.LocalConformance && !validRemoteIdentity(entries[0].ID)) {
				return fmt.Errorf("Drive bundle %s is absent, ambiguous or has conflicting size/identity", bundle.Label.ID)
			}
		}
	}
	stagePath := filepath.Join(filepath.Dir(plan.Request.To), ".pair-restore-staging")
	for _, key := range keys {
		g := groups[key]
		names := make([]string, 0, len(g.bundles))
		for _, bundle := range g.bundles {
			names = append(names, bundle.Label.ID+".tar.zst.gpg")
		}
		if err := r.Rclone.DownloadBatch(ctx, g.pin, names, filepath.Join(stagePath, "ciphertext"), filepath.Join(stagePath, "files-from"), g.bytes, g.maxBytes); err != nil {
			return err
		}
	}
	return nil
}
