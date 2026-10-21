package historyarchive

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"time"
)

// indexDriveFolder is private observation state, never an additional persisted
// evidence clock. Each folder becomes one existing complete IndexRefresh.
type indexDriveFolder struct {
	dest       DriveDestination
	pin        PinnedDrive
	relevant   map[string]RemoteEntry
	names      []string
	labelBytes int64
}

type indexDriveParent struct {
	pin      PinnedDrive
	year     bool
	children map[string]PinnedDrive
}

func indexDriveYear(name string) bool {
	if len(name) != 4 || name == "0000" {
		return false
	}
	for _, c := range name {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func indexDriveMonth(name string) bool {
	return len(name) == 2 && name[0] >= '0' && name[0] <= '1' && name[1] >= '0' && name[1] <= '9' && name >= "01" && name <= "12"
}

func (d DriveIndexSync) indexDriveChildren(ctx context.Context, pin PinnedDrive, inventory map[string][]RemoteEntry, year bool) (map[string]PinnedDrive, error) {
	children := map[string]PinnedDrive{}
	for name := range inventory {
		if year && !indexDriveYear(name) || !year && !indexDriveMonth(name) {
			continue
		}
		child, found, err := d.Rclone.ResolveChild(ctx, pin, name, inventory)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errors.New("Drive archive child disappeared from inventory")
		}
		children[name] = child
	}
	return children, nil
}

func sortedDriveChildren(children map[string]PinnedDrive) []string {
	names := make([]string, 0, len(children))
	for name := range children {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// observeIndexTree inventories every admitted folder before any download. The
// global metadata and destination limits apply across otherwise valid flats.
func (d DriveIndexSync) observeIndexTree(ctx context.Context, root DriveDestination) ([]indexDriveFolder, []indexDriveParent, error) {
	pin, err := d.Rclone.ResolveDestination(ctx, root)
	if err != nil {
		return nil, nil, err
	}
	if !validRemoteIdentity(pin.FolderID) {
		if !d.Rclone.LocalConformance {
			return nil, nil, errors.New("Drive index folder lacks stable identity")
		}
		pin.FolderID = pin.Root
	}
	inventory, err := d.Rclone.ObserveFlatInventory(ctx, pin)
	if err != nil {
		return nil, nil, err
	}
	var folders []indexDriveFolder
	var used int64
	count := 0
	add := func(dest DriveDestination, pin PinnedDrive, inventory map[string][]RemoteEntry) error {
		if len(folders) >= MaxIndexDestinations {
			return errors.New("Drive index exceeds physical destination bound")
		}
		relevant, names, e := indexDriveInventory(inventory)
		if e != nil {
			return e
		}
		if len(names) > MaxIndexRecords-count {
			return errors.New("Drive index exceeds aggregate label count")
		}
		var bytes int64
		for _, name := range names {
			bytes += relevant[name].Size
		}
		if bytes > MaxIndexBytes-used {
			return errors.New("Drive index exceeds aggregate label bytes")
		}
		used += bytes
		count += len(names)
		folders = append(folders, indexDriveFolder{dest: dest, pin: pin, relevant: relevant, names: names, labelBytes: bytes})
		return nil
	}
	if err = add(root, pin, inventory); err != nil {
		return nil, nil, err
	}
	years, err := d.indexDriveChildren(ctx, pin, inventory, true)
	if err != nil {
		return nil, nil, err
	}
	parents := []indexDriveParent{{pin: pin, year: true, children: years}}
	for _, year := range sortedDriveChildren(years) {
		yearPin := years[year]
		yearInventory, e := d.Rclone.ObserveFlatInventory(ctx, yearPin)
		if e != nil {
			return nil, nil, e
		}
		months, e := d.indexDriveChildren(ctx, yearPin, yearInventory, false)
		if e != nil {
			return nil, nil, e
		}
		if len(months) > MaxIndexDestinations-len(folders) {
			return nil, nil, errors.New("Drive index exceeds physical destination bound")
		}
		parents = append(parents, indexDriveParent{pin: yearPin, children: months})
		for _, month := range sortedDriveChildren(months) {
			monthPin := months[month]
			monthInventory, e := d.Rclone.ObserveFlatInventory(ctx, monthPin)
			if e != nil {
				return nil, nil, e
			}
			dest, e := ParseDriveDestination(root.Remote, root.Folder+"/"+year+"/"+month)
			if e != nil {
				return nil, nil, e
			}
			if e = add(dest, monthPin, monthInventory); e != nil {
				return nil, nil, e
			}
		}
	}
	return folders, parents, nil
}

func (d DriveIndexSync) confirmIndexTree(ctx context.Context, root DriveDestination, folders []indexDriveFolder, parents []indexDriveParent) error {
	// Root resolution is repeated: a successful inventory of an old pinned ID
	// cannot establish that the configured root still names that ID.
	pin, err := d.Rclone.ResolveDestination(ctx, root)
	if err != nil {
		return err
	}
	if d.Rclone.LocalConformance && !validRemoteIdentity(pin.FolderID) {
		pin.FolderID = pin.Root
	}
	if pin != folders[0].pin {
		return errors.New("Drive archive root binding changed during label observation")
	}
	rootInventory, err := d.Rclone.ObserveFlatInventory(ctx, pin)
	if err != nil {
		return err
	}
	for i, parent := range parents {
		inventory := rootInventory
		if i != 0 {
			inventory, err = d.Rclone.ObserveFlatInventory(ctx, parent.pin)
			if err != nil {
				return err
			}
		}
		children, e := d.indexDriveChildren(ctx, parent.pin, inventory, parent.year)
		if e != nil {
			return e
		}
		if !reflect.DeepEqual(children, parent.children) {
			return errors.New("Drive archive directory bindings changed during label observation")
		}
	}
	for i, folder := range folders {
		inventory := rootInventory
		if i != 0 {
			inventory, err = d.Rclone.ObserveFlatInventory(ctx, folder.pin)
			if err != nil {
				return err
			}
		}
		final, _, e := indexDriveInventory(inventory)
		if e != nil {
			return e
		}
		if !reflect.DeepEqual(final, folder.relevant) {
			return errors.New("Drive archive pairs changed during label observation")
		}
	}
	return nil
}

func indexDriveLocatorInRoot(root DriveDestination, locator string) bool {
	if locator == root.String() {
		return true
	}
	rel, ok := strings.CutPrefix(locator, root.String()+"/")
	if !ok {
		return false
	}
	year, month, ok := strings.Cut(rel, "/")
	return ok && indexDriveYear(year) && indexDriveMonth(month)
}

// Missing physical folders receive empty observations through the same marker
// and receipt precedence mechanism as observed empty folders. A failed tree
// observation never reaches this function or publication.
func missingIndexDriveFolders(old LabelIndex, root DriveDestination, folders []indexDriveFolder, at time.Time) ([]IndexRefresh, error) {
	seen := map[string]bool{}
	for _, folder := range folders {
		seen["drive\n"+folder.dest.String()] = true
	}
	known := map[string]DestinationObservation{}
	for key, marker := range old.CompleteDestinations {
		known[key] = marker
	}
	// Receipt-only locations may predate the first complete tree observation.
	for _, record := range old.Records {
		for _, loc := range record.Locations {
			if loc.Backend != "drive" || !indexDriveLocatorInRoot(root, loc.Locator) {
				continue
			}
			key := loc.Backend + "\n" + loc.Locator
			if _, complete := old.CompleteDestinations[key]; complete {
				continue
			}
			prior, ok := known[key]
			if !ok || loc.LastObserved.After(prior.ObservedAt) || loc.LastObserved.Equal(prior.ObservedAt) && loc.FolderID < prior.FolderID {
				known[key] = DestinationObservation{Backend: "drive", Locator: loc.Locator, FolderID: loc.FolderID, ObservedAt: loc.LastObserved}
			}
		}
	}
	for key := range seen {
		if _, ok := known[key]; !ok {
			known[key] = DestinationObservation{}
		}
	}
	if len(known) > MaxIndexDestinations {
		return nil, errors.New("Drive index exceeds complete destination bound")
	}
	keys := make([]string, 0, len(known))
	for key, marker := range known {
		if !seen[key] && marker.Backend == "drive" && indexDriveLocatorInRoot(root, marker.Locator) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	refreshes := make([]IndexRefresh, 0, len(keys))
	for _, key := range keys {
		marker := known[key]
		refreshes = append(refreshes, IndexRefresh{Backend: "drive", Locator: marker.Locator, FolderID: marker.FolderID, At: at, Complete: true})
	}
	return refreshes, nil
}
