package runtimebundle

import "path/filepath"

// TerminalEnvironment installs the build-time compiled profile alongside this
// binary's versioned assets. Child launch never depends on a host's tic program.
func TerminalEnvironment(dataRoot string) ([]string, error) {
	result, err := Extract(StoreInput{StoreRoot: filepath.Join(dataRoot, "runtime"), Manifest: EmbeddedManifest(), ReadAsset: EmbeddedAsset, Keep: 2})
	if err != nil {
		return nil, err
	}
	return []string{"TERM=pair-vt-256color", "TERMINFO=" + filepath.Join(result.PairHome, "terminfo")}, nil
}
