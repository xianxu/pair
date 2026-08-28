package sessioninventory

import (
	"errors"
	"fmt"
	"time"
)

var ErrReadLimit = errors.New("session inventory read exceeds limit")

// StorageRoot names one authorized native storage tree. Path is an opaque OS
// location used only by Runtime implementations; facts retain Name instead.
type StorageRoot struct {
	Agent Agent  `json:"agent"`
	Name  string `json:"name"`
	Path  string `json:"-"`
}

type FileEntry struct {
	Artifact  Artifact   `json:"artifact"`
	Size      int64      `json:"size"`
	BirthTime *time.Time `json:"birth_time"`
	ModTime   *time.Time `json:"mod_time"`
}

type ListingIssuesError struct {
	Artifacts []Artifact
}

func (e *ListingIssuesError) Error() string {
	return fmt.Sprintf("%d native storage entries rejected: %v", len(e.Artifacts), ErrPathEscape)
}

func (e *ListingIssuesError) Unwrap() error { return ErrPathEscape }

type SQLiteResult struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

// Runtime is the sole IO boundary for native session discovery and live
// evidence. Every path is resolved through an authorized StorageRoot.
type Runtime interface {
	NativeRoots(Agent) []StorageRoot
	PairDataRoot() StorageRoot
	ListFiles(StorageRoot) ([]FileEntry, error)
	ReadFile(Artifact, int64) ([]byte, error)
	ReadAt(Artifact, int64, int64) ([]byte, bool, error)
	QuerySQLite(Artifact, string, int64) (SQLiteResult, error)
	ProcessChildren() map[string][]string
	ProcessIdentity(string) string
	OpenFiles(string) []string
}

type ScanResult struct {
	Facts       []Fact
	Diagnostics []Diagnostic
}

type Scanner interface {
	Scan(Runtime) ScanResult
}

type ScannerFunc func(Runtime) ScanResult

func (scanner ScannerFunc) Scan(runtime Runtime) ScanResult { return scanner(runtime) }
