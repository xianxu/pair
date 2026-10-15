package sessioninventory

import (
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"path/filepath"
)

// QueryWorkObserver counts real body IO and decoder inputs separately. The
// observer belongs to a runtime, so independent callers never share counters.
// Callbacks may be concurrent and must be safe for that caller's workload.
type QueryWorkObserver interface {
	ReadWork(Artifact, int64)
	DecodeWork(Artifact, int64)
}

func observeDecodeWork(runtime Runtime, artifact Artifact, bytes int64) {
	if observer, ok := runtime.(interface{ DecodeWork(Artifact, int64) }); ok {
		observer.DecodeWork(artifact, bytes)
	}
}

func decodeScannerStateObserved(runtime Runtime, artifact Artifact, raw []byte) (ScannerState, error) {
	observeDecodeWork(runtime, artifact, int64(len(raw)))
	return DecodeScannerState(raw)
}

func observeRecordsDecodeWork(runtime Runtime, artifact Artifact, records []FramedJSONLRecord) {
	var bytes int64
	for _, record := range records {
		bytes += int64(len(record.Bytes))
	}
	if bytes > 0 {
		observeDecodeWork(runtime, artifact, bytes)
	}
}

// observedCatalogRuntime keeps CatalogStore's strict decoder and errors intact.
// Its successful ReadFile is immediately followed by that store's decoder.
type observedCatalogRuntime struct {
	CatalogOSRuntime
	owner    OSRuntime
	artifact Artifact
}

func (r observedCatalogRuntime) ReadFile(path string) ([]byte, error) {
	raw, err := r.CatalogOSRuntime.ReadFile(path)
	r.owner.ReadWork(r.artifact, int64(len(raw)))
	if err == nil {
		r.owner.DecodeWork(r.artifact, int64(len(raw)))
	}
	return raw, err
}

func queryCatalogArtifact(runtime Runtime) Artifact {
	paths, err := artifactpath.ResolveSelectedScope(runtime.PairDataRoot().Path)
	if err != nil {
		return Artifact{StorageRoot: runtime.PairDataRoot().Name}
	}
	return Artifact{StorageRoot: runtime.PairDataRoot().Name, RelativePath: filepath.Base(paths.SessionInventoryCatalog())}
}
