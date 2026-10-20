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
			return nil, fmt.Errorf("open local restore archive %q: %w", r.Config.Sink, err)
		}
		if local != nil {
			defer local.Close()
		}
	}
	var remote []RestoreBundlePlan
	sources := map[string]string{}
	stagedPath := func(bundle RestoreBundlePlan) string {
		return filepath.Join(filepath.Dir(plan.Request.To), ".pair-restore-staging", "ciphertext", bundle.Label.ID+".tar.zst.gpg")
	}
	for _, bundle := range plan.Bundles {
		if err = ctx.Err(); err != nil {
			return inputs, err
		}
		var source *os.File
		if local != nil {
			source, err = openRegularAt(local, bundle.BundleKey, unix.O_RDONLY)
			if err != nil && !errors.Is(err, unix.ENOENT) {
				return inputs, fmt.Errorf("open local restore bundle %s at %q: %w", bundle.Label.ID, filepath.Join(r.Config.Sink, bundle.BundleKey), err)
			}
		}
		if source == nil {
			if plan.Request.From == "local" {
				return inputs, fmt.Errorf("local bundle %s at %q is absent: %w", bundle.Label.ID, filepath.Join(r.Config.Sink, bundle.BundleKey), os.ErrNotExist)
			}
			remote = append(remote, bundle)
			if len(bundle.DriveLocations) > 0 {
				sources[bundle.Label.ID] = bundle.DriveLocations[0].Locator
			}
			continue
		}
		sources[bundle.Label.ID] = filepath.Join(r.Config.Sink, bundle.BundleKey)
		out, createErr := CreateRestoreFile(dir, bundle.Label.ID+".tar.zst.gpg", budget)
		if createErr != nil {
			source.Close()
			return inputs, fmt.Errorf("create staged ciphertext %q for bundle %s from %q: %w", stagedPath(bundle), bundle.Label.ID, sources[bundle.Label.ID], createErr)
		}
		n, hash, copyErr := fileDigest(ctx, source, out, bundle.Label.Bytes)
		copyErr = errors.Join(copyErr, source.Close(), out.Sync(), out.Close())
		if copyErr != nil {
			return inputs, fmt.Errorf("copy/hash local bundle %s from %q to %q: %w", bundle.Label.ID, sources[bundle.Label.ID], stagedPath(bundle), copyErr)
		}
		if n != bundle.Label.Bytes || hash != bundle.Label.SHA256 {
			return inputs, fmt.Errorf("local bundle %s at %q ciphertext checksum/size conflict; inspect it or explicitly select Drive", bundle.Label.ID, sources[bundle.Label.ID])
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
			return inputs, fmt.Errorf("open staged ciphertext %q for bundle %s from %q: %w", stagedPath(bundle), bundle.Label.ID, sources[bundle.Label.ID], openErr)
		}
		inputs = append(inputs, stagedRestoreInput{Plan: bundle, File: file})
		n, hash, digestErr := fileDigest(ctx, file, nil, bundle.Label.Bytes)
		if digestErr != nil {
			return inputs, fmt.Errorf("hash staged ciphertext %q for bundle %s from %q: %w", stagedPath(bundle), bundle.Label.ID, sources[bundle.Label.ID], digestErr)
		}
		if n != bundle.Label.Bytes || hash != bundle.Label.SHA256 {
			return inputs, fmt.Errorf("bundle %s staged ciphertext %q from %q checksum/size conflict", bundle.Label.ID, stagedPath(bundle), sources[bundle.Label.ID])
		}
		if _, err = file.Seek(0, io.SeekStart); err != nil {
			return inputs, fmt.Errorf("rewind staged ciphertext %q for bundle %s from %q: %w", stagedPath(bundle), bundle.Label.ID, sources[bundle.Label.ID], err)
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
		return fmt.Errorf("resolve restore Drive root %q: %w", root.String(), err)
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
		return fmt.Errorf("list restore Drive root %q (folder ID %q): %w", root.String(), pin.FolderID, err)
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
					return fmt.Errorf("resolve restore Drive child %q for location %q (parent ID %q): %w", name, g.location.Locator, g.pin.FolderID, err)
				}
				if !found {
					return fmt.Errorf("recorded Drive folder child %q for location %q is absent; run sync-index", name, g.location.Locator)
				}
				g.pin = child
				if n < len(parts)-1 {
					inventory, err = observe(child)
					if err != nil {
						return fmt.Errorf("list restore Drive parent %q for location %q (folder ID %q): %w", name, g.location.Locator, child.FolderID, err)
					}
				}
			}
		}
		if g.pin.FolderID != g.location.FolderID {
			return fmt.Errorf("recorded Drive folder %q identity changed (expected %q, observed %q); run sync-index", g.location.Locator, g.location.FolderID, g.pin.FolderID)
		}
		inventory, err := observe(g.pin)
		if err != nil {
			return fmt.Errorf("list restore Drive location %q (folder ID %q): %w", g.location.Locator, g.pin.FolderID, err)
		}
		for _, bundle := range g.bundles {
			entries := inventory[bundle.Label.ID+".tar.zst.gpg"]
			if len(entries) != 1 || entries[0].IsDir || entries[0].Size != bundle.Label.Bytes || (!r.Rclone.LocalConformance && !validRemoteIdentity(entries[0].ID)) {
				return fmt.Errorf("Drive bundle %s at %q is absent, ambiguous or has conflicting size/identity", bundle.Label.ID, g.location.Locator)
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
			return fmt.Errorf("download restore Drive location %q (folder ID %q) to %q: %w", g.location.Locator, g.pin.FolderID, filepath.Join(stagePath, "ciphertext"), err)
		}
	}
	return nil
}
