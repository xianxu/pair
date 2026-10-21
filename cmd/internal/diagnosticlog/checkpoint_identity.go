package diagnosticlog

import (
	"fmt"
	"os"
)

type checkpointMatch uint8

const (
	checkpointExact checkpointMatch = iota
	checkpointDeviceOnly
)

// compareCheckpoint compares a persisted checkpoint with an observation. A
// remount may renumber devices; it grants no authority over an uncertain tail.
// Callers validating two observations from one operation must use matches.
func compareCheckpoint(observed, recorded generation) (checkpointMatch, error) {
	switch {
	case observed.Identity.Inode == 0 || recorded.Identity.Inode == 0:
		return 0, fmt.Errorf("diagnostic checkpoint inode is unknown")
	case observed.Identity.Inode != recorded.Identity.Inode:
		return 0, fmt.Errorf("diagnostic checkpoint inode changed")
	case observed.Size != recorded.Size:
		return 0, fmt.Errorf("diagnostic checkpoint size changed")
	case !observed.ModTime.Equal(recorded.ModTime):
		return 0, fmt.Errorf("diagnostic checkpoint modification time changed")
	case observed.Identity.Device != recorded.Identity.Device:
		return checkpointDeviceOnly, nil
	default:
		return checkpointExact, nil
	}
}
func observedGeneration(st os.FileInfo) generation {
	return generation{Identity: fileIdentity(st), Size: st.Size(), ModTime: st.ModTime()}
}
func matchesPersisted(st os.FileInfo, g generation) error {
	_, err := compareCheckpoint(observedGeneration(st), g)
	return err
}

// Interrupted transactions cannot use remount equivalence to grant effects.
func matchesRecovery(st os.FileInfo, g generation, operation string) error {
	if err := matches(st, g); err != nil {
		return fmt.Errorf("diagnostic %s recovery: %w", operation, err)
	}
	return nil
}
