package sessioninventory

import "os"

// FingerprintFileInfo applies the inventory's existing platform identity and
// mutation metadata to one regular evidence file. It does not read its body.
// Callers must verify a stable before/after observation around a body read.
func FingerprintFileInfo(path string, info os.FileInfo) (ArtifactFingerprint, error) {
	if info == nil || !info.Mode().IsRegular() {
		return ArtifactFingerprint{}, ErrNotRegularArtifact
	}
	entry, err := fileEntryFor(Artifact{}, path, info)
	if err != nil {
		return ArtifactFingerprint{}, err
	}
	return fingerprintFromEntry(entry), nil
}
