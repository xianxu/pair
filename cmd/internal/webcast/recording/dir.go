package recording

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

// Sink is where a recorder's records go: a recording directory, the live
// fan-out, or both (Tee). Put is called in record order from one goroutine.
//
// Records only. Asset bodies and screenshots are not records. A body is
// stored once, through the AssetWriter the asset store (#459) persists into,
// and only its asset record goes through the Sink. A screenshot goes to
// DirWriter.PutFrame, and never to a Live, because pixels never reach viewers.
type Sink interface {
	Put(Record) error
	Close() error
}

// Tee puts every record into each sink in turn; the first error wins.
func Tee(sinks ...Sink) Sink { return tee(sinks) }

type tee []Sink

func (t tee) Put(r Record) error {
	var errs []error
	for _, s := range t {
		errs = append(errs, s.Put(r))
	}
	return errors.Join(errs...)
}

func (t tee) Close() error {
	var errs []error
	for _, s := range t {
		errs = append(errs, s.Close())
	}
	return errors.Join(errs...)
}

// AssetName is an asset body's file name under assets/.
func AssetName(hash string) string { return filepath.Join("assets", hash) }

// HashOf is the content hash an asset body is stored under.
func HashOf(body []byte) string {
	h := sha256.Sum256(body)
	return hex.EncodeToString(h[:])
}

var frameFile = regexp.MustCompile(`^frames/[0-9]+\.png$`)

// DirWriter writes a recording directory: records to their files, asset
// bodies under assets/, screenshots under frames/. Safe for one writer.
//
// Each record reaches its file as it is put, in one append (pair#493). A
// recording is read while it is still being written -- the live replay, the
// notes query -- and per-file buffers held a cast's few notes (well under a
// buffer's size) until the cast stopped, two hours later; a crash would have
// lost every file's tail. The rate is low (a 2.5 h cast: ~3,500 records), so
// a write per record costs nothing that matters.
type DirWriter struct {
	dir   string
	mu    sync.Mutex
	files map[string]*os.File
}

// NewDirWriter creates dir (it must not exist yet: a recording is written
// once).
func NewDirWriter(dir string) (*DirWriter, error) {
	if err := os.Mkdir(dir, 0o700); err != nil {
		return nil, err
	}
	for _, sub := range []string{"assets", "frames"} {
		if err := os.Mkdir(filepath.Join(dir, sub), 0o700); err != nil {
			return nil, err
		}
	}
	return &DirWriter{dir: dir, files: map[string]*os.File{}}, nil
}

// Put appends a record to the file its kind lives in.
func (w *DirWriter) Put(r Record) error {
	line, err := Encode(r)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	name := FileOf(r.Kind())
	f := w.files[name]
	if f == nil {
		var err error
		if f, err = os.OpenFile(filepath.Join(w.dir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err != nil {
			return err
		}
		w.files[name] = f
	}
	// One write per line, so a reader (or a crash) sees whole lines only.
	_, err = f.Write(append(line, '\n'))
	return err
}

// AssetWriter persists asset bodies by content hash. DirWriter is the one
// on-disk writer of a recording's assets/; the asset store (#459) decides
// what to store (dedup, resample) and persists through this.
type AssetWriter interface {
	PutAsset(body []byte) (hash string, err error)
}

// PutAsset stores a body under its hash, atomically; a body already stored
// is kept.
func (w *DirWriter) PutAsset(body []byte) (string, error) {
	hash := HashOf(body)
	path := filepath.Join(w.dir, AssetName(hash))
	if _, err := os.Stat(path); err == nil {
		return hash, nil
	}
	if err := writeFileAtomic(path, body); err != nil {
		return "", err
	}
	return hash, nil
}

// PutFrame stores a screenshot and its frame record; f.File must be
// frames/<n>.png.
func (w *DirWriter) PutFrame(f *Frame, png []byte) error {
	if !frameFile.MatchString(f.File) {
		return fmt.Errorf("frame file %q is not frames/<n>.png", f.File)
	}
	if err := writeFileAtomic(filepath.Join(w.dir, f.File), png); err != nil {
		return err
	}
	return w.Put(f)
}

// writeFileAtomic is every DirWriter body write: a temporary file beside the
// target, then a rename, so a crash never leaves a truncated file under a
// real name. A crash between the two can leave a .tmp-* file; ValidateDir
// reports it, and it goes when the recording's directory does (a recording
// is removed whole, by its owner).
func writeFileAtomic(path string, body []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(body)
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// Close closes every file.
func (w *DirWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	var errs []error
	for _, f := range w.files {
		errs = append(errs, f.Close())
	}
	w.files = map[string]*os.File{}
	return errors.Join(errs...)
}

// Recording is a recording directory read back.
type Recording struct {
	Dir     string
	Files   map[string][]byte // record files by name
	Records []Record          // events, notes and pointer, merged
	Frames  []*Frame
}

// ReadDir reads a recording directory. It decodes and merges the records but
// doesn't validate them; ValidateDir does.
func ReadDir(dir string) (*Recording, error) {
	rec := &Recording{Dir: dir, Files: map[string][]byte{}}
	for _, name := range []string{FileEvents, FileNotes, FilePointer, FileFrames} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		rec.Files[name] = data
	}
	located := map[string][]Located{}
	for name, data := range rec.Files {
		ls, bad := DecodeFile(name, data)
		if len(bad) > 0 {
			return nil, errors.New(bad[0].String())
		}
		located[name] = ls
	}
	rec.Records = Records(Merge(located))
	for _, l := range located[FileFrames] {
		rec.Frames = append(rec.Frames, l.Record.(*Frame))
	}
	return rec, nil
}

// Asset reads a stored body by hash.
func (r *Recording) Asset(hash string) ([]byte, error) { return readAsset(r.Dir, hash) }

// Asset reads back a body this writer stored, so the asset store (#459)
// serves from disk instead of keeping bodies in memory.
func (w *DirWriter) Asset(hash string) ([]byte, error) { return readAsset(w.dir, hash) }

func readAsset(dir, hash string) ([]byte, error) {
	if !ValidHash(hash) {
		return nil, fmt.Errorf("asset hash %q is not 64 lowercase hex", hash)
	}
	return os.ReadFile(filepath.Join(dir, AssetName(hash)))
}

// ValidateDir validates a recording directory: its records (Validate), then
// that every asset record's body is stored and hashes to its name, that
// assets/ holds nothing else, that every frame's screenshot exists, and that
// frames/ holds nothing but the index and those screenshots (a stray .tmp-*
// from a crash is reported in either directory).
func ValidateDir(dir string) ([]Finding, *State, error) {
	rec, err := ReadDir(dir)
	if err != nil {
		var fs []Finding
		fs = append(fs, Finding{File: dir, Msg: err.Error()})
		return fs, nil, nil
	}
	findings, st := Validate(rec.Files)
	ls, _ := DecodeFile(FileEvents, rec.Files[FileEvents])
	for _, l := range ls {
		a, ok := l.Record.(*Asset)
		if !ok || !ValidHash(a.Hash) {
			continue
		}
		body, err := rec.Asset(a.Hash)
		switch {
		case err != nil:
			findings = append(findings, Finding{File: l.File, Line: l.Line, Msg: "asset body missing: " + AssetName(a.Hash)})
		case HashOf(body) != a.Hash:
			findings = append(findings, Finding{File: AssetName(a.Hash), Msg: "body does not hash to its name"})
		case int64(len(body)) != a.Size:
			findings = append(findings, Finding{File: l.File, Line: l.Line, Msg: fmt.Sprintf("size %d, body is %d bytes", a.Size, len(body))})
		}
	}
	entries, err := os.ReadDir(filepath.Join(dir, "assets"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, nil, err
	}
	for _, e := range entries {
		if !ValidHash(e.Name()) {
			findings = append(findings, Finding{File: filepath.Join("assets", e.Name()), Msg: "not a content hash"})
		}
	}
	named := map[string]bool{}
	for _, f := range rec.Frames {
		named[f.File] = true
	}
	shots, err := os.ReadDir(filepath.Join(dir, "frames"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, nil, err
	}
	for _, e := range shots {
		if name := "frames/" + e.Name(); name != FileFrames && !named[name] {
			findings = append(findings, Finding{File: name, Msg: "not a frame's screenshot (no frame record names it)"})
		}
	}
	for _, f := range rec.Frames {
		if !frameFile.MatchString(f.File) {
			findings = append(findings, Finding{File: FileFrames, Msg: fmt.Sprintf("frame file %q is not frames/<n>.png", f.File)})
		} else if _, err := os.Stat(filepath.Join(dir, f.File)); err != nil {
			findings = append(findings, Finding{File: FileFrames, Msg: "missing " + f.File})
		}
	}
	return findings, st, nil
}
