package diagnosticlog

import (
	"bytes"
	"errors"
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

// appendChunk atomically publishes bounded byte authority before touching the
// log. Ordinary diagnostic appends recover process death, not host power loss;
// durable lifecycle operations explicitly sync before taking further effects.
// Returned n counts actual payload writes, even if metadata finalization fails.
func appendChunk(path string, s *diskState, data []byte, o Options) (n int, err error) {
	if len(data) == 0 || len(data) > maxAppendBytes {
		return 0, errors.New("invalid diagnostic append chunk")
	}
	if err := o.checkContext(); err != nil {
		return 0, err
	}
	st, err := regular(path)
	if err != nil {
		return 0, err
	}
	if err := matches(st, s.Current); err != nil {
		return 0, err
	}
	at := o.Now().UTC()
	if at.IsZero() {
		return 0, errors.New("invalid append clock")
	}
	s.Appending = &appendIntent{Before: s.Current, Data: append([]byte(nil), data...), At: at}
	if err := save(path, *s, false, o); err != nil {
		return 0, err
	}
	if err := fault(o, "append-intent"); err != nil {
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
	if err := matches(st, s.Appending.Before); err != nil {
		return 0, err
	}
	if err := o.checkContext(); err != nil {
		return 0, err
	}
	if o.appendWrite != nil {
		n, err = o.appendWrite(f, data)
	} else {
		n, err = f.Write(data)
	}
	if err != nil {
		return n, err
	}
	if n != len(data) {
		return n, io.ErrShortWrite
	}
	if err := fault(o, "append-written"); err != nil {
		return n, err
	}
	if err := recoverAppend(path, s, o, false); err != nil {
		return n, err
	}
	return n, nil
}

// Recovery commits only the observed authorized prefix, never an unwritten
// suffix. Identity/size/mtime plus exact tail are the existing coordination
// contract; this is not cryptographic integrity of the old file prefix.
func recoverAppend(path string, s *diskState, o Options, durable bool) error {
	if s.Appending == nil {
		return nil
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
		return errors.New("diagnostic append identity or size changed")
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
	return f.Sync()
}
