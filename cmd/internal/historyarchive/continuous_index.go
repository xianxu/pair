package historyarchive

import "time"

// ArchivedIndex is what the catalog already holds, per file: content hashes by
// (source identity, path), and the size and mtime they were archived at, so an
// unchanged file is skipped without rehashing. A file whose content changed is
// a new version: same path, new hash.
// pair:m5-concept pure
type ArchivedIndex struct {
	hashes map[indexKey]map[string]bool
	stamps map[indexKey]map[stamp]bool
}

type indexKey struct {
	Source SourceIdentity
	Path   string
}

type stamp struct {
	Size  int64
	Mtime time.Time
}

func NewArchivedIndex() *ArchivedIndex {
	return &ArchivedIndex{hashes: map[indexKey]map[string]bool{}, stamps: map[indexKey]map[stamp]bool{}}
}

// Add records every file of one archived manifest.
func (x *ArchivedIndex) Add(m Manifest) {
	for _, f := range m.Files {
		k := indexKey{m.Source, f.Path}
		if x.hashes[k] == nil {
			x.hashes[k], x.stamps[k] = map[string]bool{}, map[stamp]bool{}
		}
		x.hashes[k][f.SHA256] = true
		x.stamps[k][stamp{f.Size, f.Mtime.UTC()}] = true
	}
}

// Seen reports a file archived at this exact size and mtime.
func (x *ArchivedIndex) Seen(source SourceIdentity, c Candidate) bool {
	return x.stamps[indexKey{source, c.Path}][stamp{c.Size, c.Mtime.UTC()}]
}

// SeenHash reports this exact content archived under this path.
func (x *ArchivedIndex) SeenHash(source SourceIdentity, path, sha string) bool {
	return x.hashes[indexKey{source, path}][sha]
}
