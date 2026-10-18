package workbenchperftest

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var errLedgerCloneUnsupported = errors.New("ledger cloning unsupported")

type ledgerWriteCounts struct {
	PaddingBytes, RecordBytes uint64
	Clones, Patches           uint64
}

// A writer lives for one fixture. Only the immutable all-space seed is shared;
// each owner gets a private clone with its distinct records at the original EOF.
type ledgerWriter struct {
	directory, template string
	size                int
	useClones           bool
	clone               func(string, string) error
	patch               func(string, string, int64) (int, error)
	counts              ledgerWriteCounts
}

func (w *ledgerWriter) write(path, records string) error {
	if !w.useClones || w.size <= len(records) {
		return w.stream(path, records, w.size)
	}
	if w.template == "" {
		seed := filepath.Join(w.directory, "ledger-padding")
		if err := w.stream(seed, "", w.size); err != nil {
			return err
		}
		w.template = seed
	}
	if err := w.clone(w.template, path); err != nil {
		// clonefile must refuse an existing destination. Never unlink that
		// file when reporting a collision; all other partial copies are ours.
		if errors.Is(err, os.ErrExist) {
			return err
		}
		if cleanup := removeLedger(path); cleanup != nil {
			return errors.Join(err, cleanup)
		}
		if errors.Is(err, errLedgerCloneUnsupported) {
			w.useClones = false
			return w.stream(path, records, w.size)
		}
		return fmt.Errorf("clone ledger: %w", err)
	}
	w.counts.Clones++
	n, err := w.patch(path, records, int64(w.size-len(records)))
	w.counts.RecordBytes += uint64(n)
	if err == nil && n != len(records) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return errors.Join(fmt.Errorf("patch ledger: %w", err), removeLedger(path))
	}
	w.counts.Patches++
	return nil
}

func removeLedger(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (w *ledgerWriter) stream(path, records string, size int) (err error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, file.Close())
		if err != nil {
			err = errors.Join(err, removeLedger(path))
		}
	}()
	padding := 0
	if size > len(records) {
		padding = size - len(records)
	}
	chunk := strings.Repeat(" ", 64<<10)
	for padding > 0 {
		length := min(padding, len(chunk))
		n, writeErr := file.WriteString(chunk[:length])
		w.counts.PaddingBytes += uint64(n)
		if writeErr != nil {
			return writeErr
		}
		if n != length {
			return io.ErrShortWrite
		}
		padding -= n
	}
	if records != "" {
		n, writeErr := file.WriteString(records)
		w.counts.RecordBytes += uint64(n)
		if writeErr != nil {
			return writeErr
		}
		if n != len(records) {
			return io.ErrShortWrite
		}
	}
	return nil
}

func patchLedgerFile(path, records string, offset int64) (int, error) {
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return 0, err
	}
	n, err := file.WriteAt([]byte(records), offset)
	return n, errors.Join(err, file.Close())
}
