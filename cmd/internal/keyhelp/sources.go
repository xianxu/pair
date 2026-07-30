package keyhelp

import "github.com/xianxu/pair/cmd/internal/runtimebundle"

// SourceReader fetches a source file's bytes. The only IO in this package.
type SourceReader interface {
	Read(path string) ([]byte, error)
}

type embeddedSources struct{}

func (embeddedSources) Read(path string) ([]byte, error) {
	return runtimebundle.EmbeddedAsset(path)
}

// DefaultSources reads from the embedded runtime bundle, so a shipped `pair keys`
// needs no files on disk and renders exactly what the binary carries.
//
// Note the deliberate asymmetry with the tests: the drift/classification tests read
// the WORKING TREE (see drift_test.go), because cmd/internal/runtimebundle/assets/
// is gitignored and `go test` never regenerates it — classifying against the
// embedded copy would validate a stale snapshot. TestEmbeddedSourcesMatchTree ties
// the two together so a stale bundle fails by name.
func DefaultSources() SourceReader { return embeddedSources{} }

const (
	nvimInitPath     = "nvim/init.lua"
	zellijConfigPath = "zellij/config.kdl"
)
