package historyarchive

import (
	"errors"
	"strings"
)

// DriveArchiveLayout owns the physical locations of legacy and month archives.
// pair:m5-concept pure
type DriveArchiveLayout struct{ Root DriveDestination }

func (l DriveArchiveLayout) MonthDestination(label BundleLabel) (DriveDestination, error) {
	root, err := ParseDriveDestination(l.Root.Remote, l.Root.Folder)
	if err != nil || root != l.Root {
		return DriveDestination{}, errors.New("invalid archive Drive root")
	}
	if err = ValidateLabel(label); err != nil {
		return DriveDestination{}, err
	}
	return ParseDriveDestination(root.Remote, root.Folder+"/"+label.From.UTC().Format("2006/01"))
}

func (l DriveArchiveLayout) ValidateRecordedDriveLocation(label BundleLabel, locator string) (DriveDestination, error) {
	month, err := l.MonthDestination(label)
	if err != nil {
		return DriveDestination{}, err
	}
	if locator != l.Root.String() && locator != month.String() {
		return DriveDestination{}, errors.New("Drive location is outside the configured root or bundle month")
	}
	remote, folder, ok := strings.Cut(locator, ":")
	if !ok {
		return DriveDestination{}, errors.New("invalid Drive location")
	}
	return ParseDriveDestination(remote, folder)
}
