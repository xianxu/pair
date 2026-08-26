package launcher

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var ErrThreadAddressClaimed = errors.New("Pair thread address already claimed")

type SessionDeleter interface {
	DeleteSession(string) error
}

// ThreadAddressClaim is the durable O_EXCL marker that serializes every
// current Pair create flow with Couch's ThreadStore allocation. The marker is
// retained for the lifetime of the durable thread; Release is only for rolling
// back a Couch allocation whose ThreadStore claim subsequently fails.
type ThreadAddressClaim struct{ path string }

type threadAddressClaimRecord struct {
	Schema int    `json:"schema"`
	Scope  string `json:"scope"`
	Tag    string `json:"tag"`
	State  string `json:"state"`
}

func (c *ThreadAddressClaim) Release() error {
	if c == nil || c.path == "" {
		return nil
	}
	return ReleaseThreadAddressClaim(c.path)
}

func ReleaseThreadAddressClaim(path string) error {
	err := os.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// ClaimNewThreadAddress atomically establishes the address authority before it
// inspects older artifacts. Every current producer creates or adopts this
// marker before writing sidecars, so a competing creator either owns the marker
// or observes it; there is no scan-then-claim interval.
func ClaimNewThreadAddress(globalDataDir string, scope RepoScope, tag string) (*ThreadAddressClaim, error) {
	paths := NewScopedPaths(globalDataDir, scope, tag)
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		return nil, fmt.Errorf("create Pair scope: %w", err)
	}
	claim, err := createThreadAddressMarker(paths, scope, tag, "reserved")
	if err != nil {
		return nil, err
	}
	marker := paths.ThreadClaim()
	rollback := func(primary error) (*ThreadAddressClaim, error) {
		return nil, errors.Join(primary, claim.Release())
	}

	entries, err := os.ReadDir(paths.ScopeDir())
	if err != nil {
		return rollback(fmt.Errorf("scan scoped Pair artifacts: %w", err))
	}
	for _, entry := range entries {
		if entry.Name() != filepath.Base(marker) && OwnsTagArtifact(entry.Name(), tag) {
			return rollback(ErrThreadAddressClaimed)
		}
	}
	collision, err := strictSessionBindingCollision(filepath.Join(globalDataDir, "session-names.jsonl"), scope.Key, tag)
	if err != nil {
		return rollback(err)
	}
	if collision {
		return rollback(ErrThreadAddressClaimed)
	}
	return claim, nil
}

func createThreadAddressMarker(paths ScopedPaths, scope RepoScope, tag, state string) (*ThreadAddressClaim, error) {
	marker := paths.ThreadClaim()
	f, err := os.OpenFile(marker, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return nil, ErrThreadAddressClaimed
	}
	if err != nil {
		return nil, fmt.Errorf("claim Pair thread address: %w", err)
	}
	claim := &ThreadAddressClaim{path: marker}
	rollback := func(primary error) (*ThreadAddressClaim, error) {
		return nil, errors.Join(primary, claim.Release())
	}
	payload, _ := json.Marshal(threadAddressClaimRecord{Schema: 1, Scope: scope.Key, Tag: tag, State: state})
	if _, err := f.Write(append(payload, '\n')); err != nil {
		_ = f.Close()
		return rollback(fmt.Errorf("write Pair thread claim: %w", err))
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return rollback(fmt.Errorf("sync Pair thread claim: %w", err))
	}
	if err := f.Close(); err != nil {
		return rollback(fmt.Errorf("close Pair thread claim: %w", err))
	}
	return claim, nil
}

// EnsureThreadAddressForPair makes the native Pair create flow participate in
// the same O_EXCL authority as Couch before it writes any sidecar or session
// binding. Historical direct-Pair threads adopt a marker over their existing
// artifacts; a Couch child may use the marker its parent already committed.
func EnsureThreadAddressForPair(globalDataDir string, scope RepoScope, tag string, couchOwned bool) error {
	paths := NewScopedPaths(globalDataDir, scope, tag)
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		return fmt.Errorf("create Pair scope: %w", err)
	}
	if couchOwned {
		return establishReservedThreadAddress(paths, scope, tag)
	}
	claim, err := createThreadAddressMarker(paths, scope, tag, "established")
	if err == nil {
		// Direct Pair owns this durable marker; it is intentionally retained.
		_ = claim
		return nil
	}
	if !errors.Is(err, ErrThreadAddressClaimed) {
		return err
	}
	record, readErr := readThreadAddressClaim(paths.ThreadClaim())
	if readErr == nil && record.Schema == 1 && record.Scope == scope.Key && record.Tag == tag && record.State == "established" {
		return nil
	}
	return ErrThreadAddressClaimed
}

func establishReservedThreadAddress(paths ScopedPaths, scope RepoScope, tag string) error {
	return establishReservedThreadAddressWithHook(paths, scope, tag, nil)
}

func establishReservedThreadAddressWithHook(paths ScopedPaths, scope RepoScope, tag string, beforeRename func()) error {
	record, err := readThreadAddressClaim(paths.ThreadClaim())
	if err != nil {
		return fmt.Errorf("Couch thread claim missing: %w", err)
	}
	if record.Schema != 1 || record.Scope != scope.Key || record.Tag != tag || record.State != "reserved" {
		return ErrThreadAddressClaimed
	}
	record.State = "established"
	payload, _ := json.Marshal(record)
	return writeThreadAddressClaimAtomic(paths.ThreadClaim(), append(payload, '\n'), beforeRename)
}

func writeThreadAddressClaimAtomic(path string, payload []byte, beforeRename func()) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".publish-")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if beforeRename != nil {
		beforeRename()
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

// ThreadAddressEstablished reports Pair's durable registration evidence for a
// Couch-owned address. Missing/reserved is absent evidence; malformed or
// mismatched markers are errors and therefore never interpreted as free.
func ThreadAddressEstablished(globalDataDir string, scope RepoScope, tag string) (bool, error) {
	paths := NewScopedPaths(globalDataDir, scope, tag)
	if err := paths.Validate(); err != nil {
		return false, err
	}
	record, err := readThreadAddressClaim(paths.ThreadClaim())
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read Pair thread registration: %w", err)
	}
	if record.Schema != 1 || record.Scope != scope.Key || record.Tag != tag {
		return false, fmt.Errorf("Pair thread registration does not match requested address")
	}
	switch record.State {
	case "reserved":
		return false, nil
	case "established":
		return true, nil
	default:
		return false, fmt.Errorf("invalid Pair thread registration state %q", record.State)
	}
}

func readThreadAddressClaim(path string) (threadAddressClaimRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return threadAddressClaimRecord{}, err
	}
	var record threadAddressClaimRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return threadAddressClaimRecord{}, err
	}
	return record, nil
}

// QuiesceThreadSession resolves the public zellij session only through the
// durable composite-address index, then delegates deletion through the
// launcher's existing session lifecycle seam. Missing binding is proof that
// Pair had not reached the pre-session publication point; malformed or
// reassigned evidence fails closed.
func QuiesceThreadSession(globalDataDir, scopeKey, tag string, deleter SessionDeleter) error {
	if deleter == nil {
		return errors.New("quiesce Pair thread session: nil session deleter")
	}
	path := filepath.Join(globalDataDir, "session-names.jsonl")
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read Pair session-name index: %w", err)
	}
	defer f.Close()
	var index SessionNameIndex
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry SessionNameEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return fmt.Errorf("malformed Pair session-name index: %w", err)
		}
		index.Entries = append(index.Entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read Pair session-name index: %w", err)
	}
	entry, ok := index.latestFor(scopeKey, tag)
	if !ok {
		return nil
	}
	owner, owned := index.ownerOf(entry.SessionName)
	if !owned || owner.ScopeKey != scopeKey || owner.Tag != tag || !isPairSessionName(entry.SessionName) {
		return fmt.Errorf("Pair session binding for %s/%s is not exclusively owned", scopeKey, tag)
	}
	if err := deleter.DeleteSession(entry.SessionName); err != nil {
		return fmt.Errorf("delete Pair session %q: %w", entry.SessionName, err)
	}
	return nil
}

func strictSessionBindingCollision(path, scopeKey, tag string) (bool, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read Pair session-name index: %w", err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry SessionNameEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return false, fmt.Errorf("malformed Pair session-name index: %w", err)
		}
		if entry.ScopeKey == scopeKey && entry.Tag == tag {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("read Pair session-name index: %w", err)
	}
	return false, nil
}
