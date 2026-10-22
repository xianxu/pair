package sessionsync

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/durablefile"
)

// Entry is one landed delivery to a slot. Head is the body's first line,
// truncated: enough to recognize a submit (`submitted: <path>`), and no more
// message content than that is kept.
type Entry struct {
	ID   string
	From string
	At   time.Time
	// AddressedAt is broker admission time, used only to attribute renamed
	// paths. At remains the delivery horizon, display and retention authority.
	AddressedAt       time.Time `json:",omitempty"`
	Head              string
	Uncertain         bool                      `json:",omitempty"`
	Origin            couchmessage.Origin       `json:",omitempty"`
	RemoteEdited      bool                      `json:",omitempty"`
	RemoteSource      couchmessage.RemoteSource `json:",omitempty"`
	DocumentID        string                    `json:",omitempty"`
	rawFields         map[string]json.RawMessage
	rawSource         map[string]json.RawMessage
	provenancePresent bool
}

// The ledger's bound (ARCH-FUNERAL): trim before an append would cross
// trimBytes, keeping at most keepRows rows younger than keepFor. One
// file per slot address, so the store holds at most trimBytes plus one row
// per slot. A row trimmed while still unfolded stops being owed; at 14 days
// it is a dead thread, not a pending one. The size check is one stat per
// preflight and one after append; only a trim reads the file. Failed retention
// must stop growth: before adding a row that would cross trimBytes, trim first.
// A failed post-append operation can leave at most one codec-bounded row of
// overshoot; the next owner's preflight refuses before another append. Damaged
// evidence is retained for inspection, never truncated to make progress.
const (
	trimBytes = 256 << 10
	keepRows  = 512
	keepFor   = 14 * 24 * time.Hour
	headMax   = 256
	// A submit's head carries the file's absolute path, so it keeps more.
	submitHeadMax = 4096
)

// Ledger is the per-slot delivery record under the Couch store. Couch's
// message service appends (one writer per store, the supervisor); the
// `couch --session-sync` CLI reads.
type Ledger struct {
	Dir string
	Now func() time.Time
	mu  sync.Mutex
}

// LedgerDir is where a Couch store keeps the ledger.
func LedgerDir(storeDir string) string { return filepath.Join(storeDir, "session-sync") }

// ValidSlot reports whether slot is an exact Couch slot address (repo:N),
// by couch's own workspace-reference grammar: one grammar, not a copy.
func ValidSlot(slot string) bool {
	ref, matched, err := couchcore.ParseWorkspaceReference(slot)
	return matched && err == nil && ref.Repo != "" && ref.String() == slot
}

// path is the slot's ledger file plus suffix. The address is escaped into
// one safe filename component, whatever characters its family uses.
func (l *Ledger) path(slot, suffix string) (string, error) {
	if !ValidSlot(slot) {
		return "", fmt.Errorf("invalid slot address %q", slot)
	}
	return filepath.Join(l.Dir, url.QueryEscape(slot)+suffix), nil
}

// Head truncates a message body to the line a ledger row keeps. A submit
// keeps its whole path line (up to submitHeadMax), so recognizing it never
// depends on how deep the file lives.
func Head(body string) string {
	head, _, _ := strings.Cut(body, "\n")
	limit := headMax
	if strings.HasPrefix(head, SubmitPrefix) {
		limit = submitHeadMax
	}
	if len(head) > limit {
		head = head[:limit]
	}
	return head
}

// Append records a delivery to slot.
func (l *Ledger) Append(slot string, e Entry) error {
	p, err := l.path(slot, ".jsonl")
	if err != nil {
		return err
	}
	e.Head = Head(e.Head)
	line, err := encodeLedgerEntry(e)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := os.MkdirAll(l.Dir, 0o700); err != nil {
		return err
	}
	if err := l.trim(p, len(line)+1); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	_, werr := f.Write(append(line, '\n'))
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return werr
	}
	if err := l.expireGap(slot); err != nil {
		return err
	}
	return l.trim(p, 0)
}

// trim rewrites a file past trimBytes, atomically, so a concurrent reader
// sees the old file or the new one. Append opens the file per call, so the
// next append lands in the new inode. Reserve includes the next encoded row
// during preflight; zero is the post-append size check. Both use the same read,
// validation and atomic publication path, so a failure cannot admit more data.
func (l *Ledger) trim(p string, reserve int) error {
	info, err := os.Stat(p)
	if err != nil || info.Size()+int64(reserve) <= trimBytes {
		return err
	}
	rows, err := readRows(p)
	if err != nil {
		return err
	}
	cutoff := l.now().Add(-keepFor)
	var kept []Entry
	for _, r := range rows {
		if !r.At.Before(cutoff) {
			kept = append(kept, r)
		}
	}
	// Newest first, until the row cap or half the size bound: the result is
	// always well under trimBytes, so a trim never repeats on the next append.
	var lines [][]byte
	size := 0
	for i := len(kept) - 1; i >= 0 && len(lines) < keepRows; i-- {
		line, err := encodeLedgerEntry(kept[i])
		if err != nil {
			return err
		}
		if size+len(line)+1 > trimBytes/2 {
			break
		}
		size += len(line) + 1
		lines = append(lines, line)
	}
	var b strings.Builder
	for i := len(lines) - 1; i >= 0; i-- {
		b.Write(lines[i])
		b.WriteByte('\n')
	}
	return writeAtomic(p, []byte(b.String()))
}

func (l *Ledger) now() time.Time {
	if l.Now != nil {
		return l.Now()
	}
	return time.Now()
}

// Read returns slot's rows, oldest first. A missing file is no deliveries.
func (l *Ledger) Read(slot string) ([]Entry, error) {
	p, err := l.path(slot, ".jsonl")
	if err != nil {
		return nil, err
	}
	return readRows(p)
}

// readRows tolerates legacy torn tails. A recognized provenance row that fails
// validation refuses the read; it can never silently become clean local evidence.
func readRows(p string) ([]Entry, error) {
	f, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var rows []Entry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		var e Entry
		if err := e.UnmarshalJSON(sc.Bytes()); err != nil {
			if hasLedgerProvenance(sc.Bytes()) {
				return nil, fmt.Errorf("invalid delivery provenance: %w", err)
			}
			continue
		}
		if e.ID != "" {
			rows = append(rows, e)
		}
	}
	return rows, sc.Err()
}

// writeAtomic publishes an agent-private file durably (temp, fsync, rename).
func writeAtomic(p string, data []byte) error {
	return durablefile.WriteAtomic(p, data, "."+filepath.Base(p)+".tmp-*")
}

// replaceShared publishes the operator's file atomically, so a reader outside
// the flock (the operator's editor reloading a clean buffer) sees the old or
// the new content, never a truncated one. Parley and nvim notice the change by
// the path's mtime, which a rename moves too. It writes through a symlink to
// its target, so the link survives, and keeps the file's mode. A rename still
// breaks a hard link and drops a foreign owner, xattrs and ACLs; a session-sync
// file is one path owned by the operator, so that is accepted.
func replaceShared(p string, data []byte) error {
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		return err
	}
	info, err := os.Stat(real)
	if err != nil {
		return err
	}
	return durablefile.WriteAtomicMode(real, data, "."+filepath.Base(real)+".tmp-*", info.Mode().Perm())
}
