package sessionsync

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Entry is one landed delivery to a slot. Head is the body's first line,
// truncated: enough to recognize a submit (`submitted: <path>`), and no more
// message content than that is kept.
type Entry struct {
	ID        string
	From      string
	At        time.Time
	Head      string
	Uncertain bool `json:",omitempty"`
}

// The ledger's bound (ARCH-FUNERAL): a slot's file is trimmed when it grows
// past trimBytes, keeping at most keepRows rows younger than keepFor. One
// file per slot address, so the store holds at most trimBytes plus one row
// per slot. A row trimmed while still unfolded stops being owed; at 14 days
// it is a dead thread, not a pending one. The size check is one stat per
// append; only a trim reads the file.
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

var slotAddress = regexp.MustCompile(`^[A-Za-z0-9._-]+:[0-9]+$`)

func (l *Ledger) path(slot string) (string, error) {
	if !slotAddress.MatchString(slot) || strings.HasPrefix(slot, ".") {
		return "", fmt.Errorf("invalid slot address %q", slot)
	}
	return filepath.Join(l.Dir, strings.Replace(slot, ":", "_", 1)+".jsonl"), nil
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
	p, err := l.path(slot)
	if err != nil {
		return err
	}
	e.Head = Head(e.Head)
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := os.MkdirAll(l.Dir, 0o700); err != nil {
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
	return l.trim(p)
}

// trim rewrites a file past trimBytes, atomically, so a concurrent reader
// sees the old file or the new one. Append opens the file per call, so the
// next append lands in the new inode.
func (l *Ledger) trim(p string) error {
	info, err := os.Stat(p)
	if err != nil || info.Size() <= trimBytes {
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
	if len(kept) > keepRows {
		kept = kept[len(kept)-keepRows:]
	}
	var b strings.Builder
	for _, r := range kept {
		line, err := json.Marshal(r)
		if err != nil {
			return err
		}
		b.Write(line)
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
	p, err := l.path(slot)
	if err != nil {
		return nil, err
	}
	return readRows(p)
}

// readRows skips a line that does not decode: the writer appends whole lines,
// so only a torn tail can fail, and it is the newest row's write in flight.
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
		if json.Unmarshal(sc.Bytes(), &e) == nil && e.ID != "" {
			rows = append(rows, e)
		}
	}
	return rows, sc.Err()
}

func writeAtomic(p string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(p), "."+filepath.Base(p)+".tmp-*")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(data)
	if cerr := tmp.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Rename(tmp.Name(), p)
	}
	if werr != nil {
		_ = os.Remove(tmp.Name())
	}
	return werr
}
