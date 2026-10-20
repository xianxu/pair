package historyarchive

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

const DefaultDriveMaxFiles = 20000

// DriveInventoryObservation records a file count at a configured Drive root.
// pair:m5-concept pure
type DriveInventoryObservation struct {
	Destination string `json:"destination"`
	FolderID    string `json:"folder_id"`
	Phase       string `json:"phase"`
	Files       int    `json:"files"`
}

// ParseDriveMaxFiles parses the optional positive file-warning threshold.
func ParseDriveMaxFiles(raw string) (int, error) {
	if raw == "" {
		return DefaultDriveMaxFiles, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 0, errors.New("PAIR_ARCHIVE_DRIVE_MAX_FILES must be a positive integer")
	}
	return limit, nil
}

// observeDriveInventory counts objects, retaining directories only in the total
// used by the destination's separate entry-capacity accounting.
func observeDriveInventory(destination DriveDestination, folderID, phase string, inventory map[string][]RemoteEntry) (DriveInventoryObservation, int) {
	observation := DriveInventoryObservation{Destination: destination.String(), FolderID: folderID, Phase: phase}
	total := 0
	for _, entries := range inventory {
		total += len(entries)
		for _, entry := range entries {
			if !entry.IsDir {
				observation.Files++
			}
		}
	}
	return observation, total
}

func (o *DriveInventoryObservation) Line() string {
	if o == nil {
		return "Drive file count unavailable"
	}
	return fmt.Sprintf("Drive file count: %d at %s (root ID %s; %s)", o.Files, driveCountText(o.Destination, 160), driveCountText(o.FolderID, 96), driveCountText(o.Phase, 24))
}

func (o *DriveInventoryObservation) Warning(limit int) string {
	if o == nil || limit <= 0 || o.Files <= limit {
		return ""
	}
	return fmt.Sprintf("%s exceeds file threshold %d. Ask the hub to arrange year subfolders on Drive.", o.Line(), limit)
}

// driveCountText also strips invisible format controls before bounding display
// text. Persisted identities remain unchanged.
func driveCountText(value string, limit int) string {
	value = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Cf, r) {
			return ' '
		}
		return r
	}, value)
	return compactDatabaseText(value, limit)
}

func validateDriveInventoryObservation(o *DriveInventoryObservation) error {
	if o == nil {
		return nil
	}
	remote, folder, ok := strings.Cut(o.Destination, ":")
	if !ok || remote == "" || folder == "" {
		return errors.New("invalid Drive inventory destination")
	}
	destination, err := ParseDriveDestination(remote, folder)
	if err != nil || destination.String() != o.Destination {
		return errors.New("invalid Drive inventory destination")
	}
	if !validRemoteIdentity(o.FolderID) || o.Files < 0 {
		return errors.New("invalid Drive inventory folder or file count")
	}
	if o.Phase != "before" && o.Phase != "after" {
		return errors.New("invalid Drive inventory phase")
	}
	return nil
}
