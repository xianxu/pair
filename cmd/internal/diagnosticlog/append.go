package diagnosticlog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"time"
)

const maxAppendBytes = 64 << 10

type appendIntent struct {
	Before generation `json:"before"`
	Data   []byte     `json:"data"`
	At     time.Time  `json:"at"`
}

// appendChunk reserves a bounded digest without duplicating payload. A healthy
// batch publishes one state checkpoint; optional hot appends do not fsync.
func appendChunk(path string, s *diskState, data []byte, o Options) (n int, err error) {
	if len(data) == 0 || len(data) > maxAppendBytes {
		return 0, errors.New("invalid diagnostic append chunk")
	}
	if err = o.checkContext(); err != nil {
		return 0, err
	}
	st, err := regular(path)
	if err != nil {
		return 0, err
	}
	if err = matches(st, s.Current); err != nil {
		return 0, err
	}
	at := o.Now().UTC()
	if err = writeReservation(path, s.Current, data, at, o); err != nil {
		return 0, err
	}
	if err = fault(o, "append-intent"); err != nil {
		return 0, err
	}
	f, err := openRegular(path, syscall.O_WRONLY|syscall.O_APPEND)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	st, err = f.Stat()
	if err != nil {
		return 0, err
	}
	if err = matches(st, s.Current); err != nil {
		return 0, err
	}
	if err = o.checkContext(); err != nil {
		return 0, err
	}
	if o.appendWrite != nil {
		n, err = o.appendWrite(f, data)
	} else {
		n, err = f.Write(data)
	}
	o.observe("write", path, n)
	if n < 0 || n > len(data) {
		return 0, errors.New("invalid diagnostic write progress")
	}
	if n != len(data) && err == nil {
		err = io.ErrShortWrite
	}
	if n == 0 {
		return n, err
	}
	if e := fault(o, "append-written"); e != nil {
		return n, errors.Join(err, e)
	}
	if n != len(data) {
		// The live writer knows the actual prefix. A killed writer cannot invent
		// this shortened digest; its uncertain partial effect remains untouched.
		if e := writeReservation(path, s.Current, data[:n], at, o); e != nil {
			return n, errors.Join(err, e)
		}
	}
	return n, errors.Join(err, recoverReservedAppend(path, s, o, false))
}

// Recovery commits only the observed authorized prefix, never an unwritten
// suffix. Identity/size/mtime plus exact tail are the existing coordination
// contract; this is not cryptographic integrity of the old file prefix.
func recoverAppend(path string, s *diskState, o Options, durable bool) error {
	if s.Appending == nil {
		return recoverReservedAppend(path, s, o, durable)
	}
	if err := o.checkContext(); err != nil {
		return err
	}
	intent := s.Appending
	f, err := openRegular(path, syscall.O_RDWR)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	written := st.Size() - intent.Before.Size
	if fileIdentity(st) != intent.Before.Identity || written < 0 || written > int64(len(intent.Data)) {
		return fmt.Errorf("diagnostic legacy-append recovery: %w", matches(st, intent.Before))
	}
	next := *s
	next.Appending = nil
	if written == 0 {
		if err := matches(st, intent.Before); err != nil {
			return err
		}
	} else {
		if err := o.checkContext(); err != nil {
			return err
		}
		tail := make([]byte, int(written))
		if _, err := io.ReadFull(io.NewSectionReader(f, intent.Before.Size, written), tail); err != nil {
			return err
		}
		if !bytes.Equal(tail, intent.Data[:int(written)]) {
			return errors.New("diagnostic append tail changed")
		}
		next.Current = intent.Before
		next.Current.Size = st.Size()
		next.Current.ModTime = st.ModTime()
		next.Current.LastWrite = intent.At
		verify, err := f.Stat()
		if err != nil {
			return err
		}
		if err := matches(verify, next.Current); err != nil {
			return err
		}
		if err := o.checkContext(); err != nil {
			return err
		}
		if durable {
			if err := syncAppendPayload(f, o); err != nil {
				return err
			}
		}
		if err := fault(o, "append-verified"); err != nil {
			return err
		}
	}
	if err := save(path, next, durable, o); err != nil {
		return err
	}
	*s = next
	return fault(o, "append-committed")
}

func syncAppendPayload(f *os.File, o Options) error {
	if err := o.checkContext(); err != nil {
		return err
	}
	if o.appendSync != nil {
		return o.appendSync(f)
	}
	o.observe("sync", f.Name(), 0)
	return f.Sync()
}
