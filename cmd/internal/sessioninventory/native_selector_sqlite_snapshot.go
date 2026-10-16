package sessioninventory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/csv"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// NativeSelectorSQLiteSnapshotSource keeps snapshot reads inside the attempt's
// existing recovery budget and descriptor-checked native IO.
type NativeSelectorSQLiteSnapshotSource interface {
	StatRecoveryFile(context.Context, Artifact) (FileEntry, error)
	ReadRecoveryAt(context.Context, FileEntry, int64, int64) ([]byte, bool, error)
	ChargeNativeSelectorSQLiteBytes(int64) error
}

// NativeSelectorSQLiteSnapshot owns private artifacts until Close. Query never
// opens the native database, WAL or shared-memory sidecar through SQLite.
type NativeSelectorSQLiteSnapshot interface {
	Query(context.Context, string, int64) (SQLiteResult, error)
	Validate(context.Context) error
	Close() error
}

type nativeSelectorSQLiteSnapshotFactory interface {
	NewNativeSelectorSQLiteSnapshot(context.Context, Artifact, NativeSelectorSQLiteSnapshotSource, int64) (NativeSelectorSQLiteSnapshot, error)
}

func isCodexSelectorSQLite(a Artifact) bool {
	return a.StorageRoot == "codex-state" && a.RelativePath == "state_5.sqlite"
}

type nativeSQLiteSnapshot struct {
	source          NativeSelectorSQLiteSnapshotSource
	entries         []FileEntry
	journal         Artifact
	directory, path string
	walHeader       []byte
}

// NewNativeSelectorSQLiteSnapshot supports only an observed stable present WAL.
// Copy WAL first, then DB, and fully verify both source byte streams again.
// SQLite WAL resets change generation salts even if inode/size/mtime repeat;
// same-generation checkpoints copy only pages already covered by that WAL.
// See sqlite.org/fileformat.html sections 4.3-4.5. Private quick_check checks
// recovery validity, while source fences and full comparisons check coherence.
func (r OSRuntime) NewNativeSelectorSQLiteSnapshot(ctx context.Context, a Artifact, source NativeSelectorSQLiteSnapshotSource, remaining int64) (result NativeSelectorSQLiteSnapshot, resultErr error) {
	if ctx == nil || source == nil || !isCodexSelectorSQLite(a) {
		return nil, ErrRootUnknown
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	db, err := source.StatRecoveryFile(ctx, a)
	if err != nil {
		return nil, err
	}
	walArtifact := a
	walArtifact.RelativePath += "-wal"
	wal, err := source.StatRecoveryFile(ctx, walArtifact)
	if err != nil {
		return nil, err
	}
	if db.Size < 100 || wal.Size < 32 || db.Size > remaining/2 || wal.Size > remaining/2-db.Size {
		return nil, ErrReadLimit
	}
	journal := a
	journal.RelativePath += "-journal"
	if _, err = source.StatRecoveryFile(ctx, journal); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			err = ErrArtifactChanged
		}
		return nil, err
	}
	parent, err := r.nativeSelectorSQLiteScratchParent()
	if err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp(parent, "pair-native-selector-sqlite-")
	if err != nil {
		return nil, err
	}
	snapshot := &nativeSQLiteSnapshot{source: source, entries: []FileEntry{wal, db}, journal: journal, directory: directory, path: filepath.Join(directory, "state_5.sqlite")}
	ready := false
	defer func() {
		if !ready {
			if closeErr := snapshot.Close(); closeErr != nil {
				resultErr = errors.Join(resultErr, closeErr)
			}
		}
	}()
	// Order is significant: the stable WAL generation brackets the main DB copy.
	for _, entry := range snapshot.entries {
		suffix := ""
		if entry.Artifact.RelativePath == walArtifact.RelativePath {
			suffix = "-wal"
		}
		path := snapshot.path + suffix
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return nil, err
		}
		_, copyErr := snapshot.readSource(ctx, entry, file)
		closeErr := file.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
	}
	walFile, err := os.Open(snapshot.path + "-wal")
	if err != nil {
		return nil, err
	}
	snapshot.walHeader = make([]byte, 32)
	_, headerErr := io.ReadFull(walFile, snapshot.walHeader)
	closeErr := walFile.Close()
	if headerErr != nil {
		return nil, headerErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if err = snapshot.Validate(ctx); err != nil {
		return nil, err
	}
	// Content verification catches checkpoint/recycle writes even with unchanged
	// metadata. Each second-pass byte is charged to the same source-read budget.
	for _, entry := range snapshot.entries {
		suffix := ""
		if entry.Artifact.RelativePath == walArtifact.RelativePath {
			suffix = "-wal"
		}
		file, err := os.Open(snapshot.path + suffix)
		if err != nil {
			return nil, err
		}
		copied := sha256.New()
		_, hashErr := io.Copy(copied, file)
		closeErr := file.Close()
		if hashErr != nil {
			return nil, hashErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		current, err := snapshot.readSource(ctx, entry, io.Discard)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(copied.Sum(nil), current) {
			return nil, ErrArtifactChanged
		}
	}
	if err = snapshot.Validate(ctx); err != nil {
		return nil, err
	}
	if err = snapshot.validateFormat(ctx); err != nil {
		return nil, err
	}
	check, err := snapshot.query(ctx, "PRAGMA quick_check", 4096)
	if err != nil {
		return nil, err
	}
	if len(check.Rows) != 1 || len(check.Rows[0]) != 1 || check.Rows[0][0] != "ok" {
		return nil, ErrArtifactChanged
	}
	if err = snapshot.Validate(ctx); err != nil {
		return nil, err
	}
	ready = true
	return snapshot, nil
}
func (s *nativeSQLiteSnapshot) readSource(ctx context.Context, entry FileEntry, destination io.Writer) ([]byte, error) {
	hash := sha256.New()
	writer := io.MultiWriter(destination, hash)
	for offset := int64(0); offset < entry.Size; {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		count := readChunkSize
		if count > entry.Size-offset {
			count = entry.Size - offset
		}
		raw, _, err := s.source.ReadRecoveryAt(ctx, entry, offset, count)
		if err != nil {
			return nil, err
		}
		if int64(len(raw)) != count {
			return nil, ErrArtifactChanged
		}
		if _, err = writer.Write(raw); err != nil {
			return nil, err
		}
		offset += count
	}
	return hash.Sum(nil), nil
}
func (s *nativeSQLiteSnapshot) Validate(ctx context.Context) error {
	if s.directory == "" {
		return ErrArtifactChanged
	}
	for _, entry := range s.entries {
		current, err := s.source.StatRecoveryFile(ctx, entry.Artifact)
		if err != nil {
			return err
		}
		if !sameRecoveryEntry(entry, current) {
			return ErrArtifactChanged
		}
	}
	if _, err := s.source.StatRecoveryFile(ctx, s.journal); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return ErrArtifactChanged
		}
		return err
	}
	// Re-read generation after ALL preceding source reads, including the second
	// full DB verification. Metadata alone misses a recycled same-size WAL.
	if len(s.walHeader) > 0 {
		header, _, err := s.source.ReadRecoveryAt(ctx, s.entries[0], 0, 32)
		if err != nil {
			return err
		}
		if !bytes.Equal(header, s.walHeader) {
			return ErrArtifactChanged
		}
	}
	return ctx.Err()
}
func (s *nativeSQLiteSnapshot) validateFormat(ctx context.Context) error {
	db, err := os.Open(s.path)
	if err != nil {
		return err
	}
	var header [100]byte
	_, err = io.ReadFull(db, header[:])
	closeErr := db.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if !bytes.Equal(header[:16], []byte("SQLite format 3\x00")) || header[18] != 2 || header[19] != 2 {
		return ErrArtifactChanged
	}
	wal, err := os.Open(s.path + "-wal")
	if err != nil {
		return err
	}
	var walHeader [32]byte
	_, err = io.ReadFull(wal, walHeader[:])
	if err != nil {
		_ = wal.Close()
		return err
	}
	defer wal.Close()
	magic := binary.BigEndian.Uint32(walHeader[:4])
	pageSize := uint32(binary.BigEndian.Uint16(header[16:18]))
	if pageSize == 1 {
		pageSize = 65536
	}
	if (magic != 0x377f0682 && magic != 0x377f0683) || binary.BigEndian.Uint32(walHeader[4:8]) != 3007000 || binary.BigEndian.Uint32(walHeader[8:12]) != pageSize || pageSize < 512 || pageSize > 65536 || pageSize&(pageSize-1) != 0 {
		return ErrArtifactChanged
	}
	bigEndian := magic == 0x377f0683
	first, second := nativeWALChecksum(walHeader[:24], bigEndian, 0, 0)
	if first != binary.BigEndian.Uint32(walHeader[24:28]) || second != binary.BigEndian.Uint32(walHeader[28:32]) {
		return ErrArtifactChanged
	}
	frame := make([]byte, 24+int(pageSize))
	committed := false
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, err := io.ReadFull(wal, frame)
		if err == io.EOF {
			break
		}
		if err != nil {
			return ErrArtifactChanged
		}
		// A reused WAL can retain complete frames from an earlier salt generation.
		// Native recovery stops there; those frames do not belong to this cohort.
		if !bytes.Equal(frame[8:16], walHeader[16:24]) {
			break
		}
		if binary.BigEndian.Uint32(frame[:4]) == 0 {
			return ErrArtifactChanged
		}
		first, second = nativeWALChecksum(frame[:8], bigEndian, first, second)
		first, second = nativeWALChecksum(frame[24:], bigEndian, first, second)
		if first != binary.BigEndian.Uint32(frame[16:20]) || second != binary.BigEndian.Uint32(frame[20:24]) {
			return ErrArtifactChanged
		}
		if binary.BigEndian.Uint32(frame[4:8]) != 0 {
			committed = true
		}
	}
	if !committed {
		return ErrArtifactChanged
	}
	return nil
}
func (s *nativeSQLiteSnapshot) Query(ctx context.Context, query string, limit int64) (SQLiteResult, error) {
	if err := s.Validate(ctx); err != nil {
		return SQLiteResult{}, err
	}
	result, err := s.query(ctx, query, limit)
	if err != nil {
		return result, err
	}
	return result, s.Validate(ctx)
}
func (s *nativeSQLiteSnapshot) query(ctx context.Context, query string, limit int64) (SQLiteResult, error) {
	if err := ctx.Err(); err != nil {
		return SQLiteResult{}, err
	}
	if s.directory == "" || limit <= 0 || limit > 1<<20 {
		return SQLiteResult{}, ErrReadLimit
	}
	stdout, stderr := newBoundedBuffer(limit), newBoundedBuffer(8192)
	// -init substitutes for default rc loading; -safe forbids external file IO
	// functions/ATTACH/extensions even when a persisted schema contains a view.
	command := exec.CommandContext(ctx, "sqlite3", "-batch", "-init", os.DevNull, "-safe", "-readonly", "-csv", "-header", s.path, query)
	// exec.Cmd keeps the last duplicate environment key. Override inherited
	// SQLite temporary storage without changing the parent process environment.
	command.Env = append(os.Environ(), "SQLITE_TMPDIR="+s.directory, "TMPDIR="+s.directory)
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return SQLiteResult{}, err
	}
	if stdout.exceeded || stderr.exceeded {
		return SQLiteResult{}, ErrReadLimit
	}
	if err := s.source.ChargeNativeSelectorSQLiteBytes(int64(len(stdout.Bytes()))); err != nil {
		return SQLiteResult{}, err
	}
	rows, err := csv.NewReader(bytes.NewReader(stdout.Bytes())).ReadAll()
	if err != nil {
		return SQLiteResult{}, err
	}
	if len(rows) == 0 {
		return SQLiteResult{}, nil
	}
	return SQLiteResult{Columns: rows[0], Rows: rows[1:]}, nil
}
func (s *nativeSQLiteSnapshot) Close() error {
	if s.directory == "" {
		return nil
	}
	directory := s.directory
	if err := os.RemoveAll(directory); err != nil {
		return err
	}
	s.directory = ""
	return nil
}

// SQLite fileformat section 4.2: checksum words follow WAL magic byte-order;
// stored checksums always use big-endian encoding.
func nativeWALChecksum(raw []byte, bigEndian bool, first, second uint32) (uint32, uint32) {
	var order binary.ByteOrder = binary.LittleEndian
	if bigEndian {
		order = binary.BigEndian
	}
	for len(raw) > 0 {
		first += order.Uint32(raw[:4]) + second
		second += order.Uint32(raw[4:8]) + first
		raw = raw[8:]
	}
	return first, second
}

// Choosing scratch is itself an observation: never create temporary children
// inside native storage, even when TMPDIR reaches it through a logical alias.
func (r OSRuntime) nativeSelectorSQLiteScratchParent() (string, error) {
	parent := os.Getenv("TMPDIR")
	if parent == "" {
		parent = os.TempDir()
	}
	if !filepath.IsAbs(parent) {
		return "", ErrPathEscape
	}
	physical, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(physical)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", ErrNotRegularArtifact
	}
	for _, root := range r.NativeRoots(AgentCodex) {
		boundary := root.Path
		if root.Name == "codex-sessions" {
			boundary = filepath.Dir(boundary)
		}
		if !filepath.IsAbs(boundary) {
			return "", ErrPathEscape
		}
		canonical, err := filepath.EvalSymlinks(boundary)
		if err != nil {
			return "", err
		}
		relative, err := filepath.Rel(canonical, physical)
		if err != nil {
			return "", err
		}
		if relative == "." || (!filepath.IsAbs(relative) && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
			return "", ErrPathEscape
		}
	}
	return physical, nil
}
