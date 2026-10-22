package artifactpath

import (
	"errors"
	"path/filepath"
)

// EditorEndpointFiles names the single owner's snapshot and reusable stage.
// These contain endpoint metadata, never editor content or listener ownership.
type EditorEndpointFiles struct{ state, stage string }

func (p EditorEndpointFiles) State() string { return p.state }
func (p EditorEndpointFiles) Stage() string { return p.stage }

func EditorEndpoints(namespace string) (EditorEndpointFiles, error) {
	if !filepath.IsAbs(namespace) || filepath.Clean(namespace) != namespace {
		return EditorEndpointFiles{}, errors.New("editor endpoint namespace must be canonical and absolute")
	}
	return EditorEndpointFiles{state: filepath.Join(namespace, "editor-endpoints.json"), stage: filepath.Join(namespace, "editor-endpoints.stage")}, nil
}
