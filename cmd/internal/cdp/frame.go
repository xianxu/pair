package cdp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// MaxFrame bounds one message; Chrome's largest legitimate frames
// (screenshots, DOM snapshots) stay well under it.
const MaxFrame = 16 << 20

// maxFrame is MaxFrame as a var so tests can exercise the limit cheaply.
var maxFrame = MaxFrame

var ErrFrameTooLarge = errors.New("cdp: frame exceeds size limit")

// Message is every CDP wire shape at once: a command (ID+Method), a reply
// (ID+Result or Error) or an event (Method without ID).
type Message struct {
	ID        int64           `json:"id,omitempty"`
	Method    string          `json:"method,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *Error          `json:"error,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
}

// Error is a CDP error reply.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

func (e *Error) Error() string {
	if e.Data != "" {
		return fmt.Sprintf("cdp: %s (%d): %s", e.Message, e.Code, e.Data)
	}
	return fmt.Sprintf("cdp: %s (%d)", e.Message, e.Code)
}

// UnmarshalJSON accepts a non-string data field verbatim: Chrome sends a
// string, but an odd error payload must not fail the whole connection as
// malformed.
func (e *Error) UnmarshalJSON(b []byte) error {
	var w struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	e.Code, e.Message, e.Data = w.Code, w.Message, ""
	if len(w.Data) > 0 && !bytes.Equal(w.Data, []byte("null")) {
		if err := json.Unmarshal(w.Data, &e.Data); err != nil {
			e.Data = string(w.Data)
		}
	}
	return nil
}

func encode(m Message) ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return append(b, 0), nil
}

type frameReader struct {
	br  *bufio.Reader
	max int
}

func newFrameReader(r io.Reader, max int) *frameReader {
	return &frameReader{br: bufio.NewReaderSize(r, 64<<10), max: max}
}

// next returns the next frame without its NUL. It accumulates ReadSlice
// chunks so an oversized frame fails once it passes max instead of being
// buffered whole. A clean EOF between frames is io.EOF; EOF mid-frame is
// io.ErrUnexpectedEOF.
func (f *frameReader) next() ([]byte, error) {
	var acc []byte
	for {
		chunk, err := f.br.ReadSlice(0)
		switch {
		case err == nil:
			chunk = chunk[:len(chunk)-1]
			if len(acc)+len(chunk) > f.max {
				return nil, ErrFrameTooLarge
			}
			return append(acc, chunk...), nil
		case errors.Is(err, bufio.ErrBufferFull):
			if len(acc)+len(chunk) > f.max {
				return nil, ErrFrameTooLarge
			}
			acc = append(acc, chunk...)
		case errors.Is(err, io.EOF):
			if len(acc)+len(chunk) > f.max {
				return nil, ErrFrameTooLarge
			}
			if len(acc)+len(chunk) > 0 {
				return nil, fmt.Errorf("cdp: truncated frame: %w", io.ErrUnexpectedEOF)
			}
			return nil, io.EOF
		default:
			return nil, err
		}
	}
}
