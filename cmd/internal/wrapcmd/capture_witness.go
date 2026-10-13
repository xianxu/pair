package wrapcmd

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

// PublishCaptureWitness runs only on two freshly truncated open files, before
// accepting child output. Every open receives a distinct capture generation.
func PublishCaptureWitness(raw, events *os.File, w artifactpath.CaptureWitness) error {
	if raw == nil || events == nil {
		return errors.New("capture files unavailable")
	}
	id := func(f *os.File) (artifactpath.CaptureFileID, error) {
		s, e := f.Stat()
		if e != nil {
			return artifactpath.CaptureFileID{}, e
		}
		n, ok := s.Sys().(*syscall.Stat_t)
		if !ok || !s.Mode().IsRegular() || s.Size() != 0 {
			return artifactpath.CaptureFileID{}, errors.New("capture is not a fresh regular file")
		}
		offset, e := f.Seek(0, 1)
		if e != nil || offset != 0 {
			return artifactpath.CaptureFileID{}, errors.New("capture offset is not zero")
		}
		return artifactpath.CaptureFileID{Device: uint64(n.Dev), Inode: uint64(n.Ino)}, nil
	}
	var e error
	if w.RawFile, e = id(raw); e != nil {
		return e
	}
	if w.EventsFile, e = id(events); e != nil {
		return e
	}
	var generation [16]byte
	if _, e = rand.Read(generation[:]); e != nil {
		return e
	}
	w.Type = "capture_witness"
	w.Version = 1
	w.Offset = 0
	w.Generation = hex.EncodeToString(generation[:])
	if e = artifactpath.ValidateCaptureWitness(w); e != nil {
		return e
	}
	data, e := json.Marshal(w)
	if e != nil {
		return e
	}
	data = append(data, '\n')
	if _, e = events.Write(data); e == nil {
		e = events.Sync()
	}
	if e != nil {
		_ = events.Truncate(0)
		_, _ = events.Seek(0, 0)
	}
	return e
}

// openWitnessedCapture invalidates old authority before changing raw bytes.
func openWitnessedCapture(rawPath, eventsPath string, getenv func(string) string) (*os.File, *os.File, error) {
	events, e := os.OpenFile(eventsPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if e != nil {
		return nil, nil, e
	}
	raw, e := os.OpenFile(rawPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if e != nil {
		events.Close()
		return nil, nil, e
	}
	ordinal, e := strconv.ParseUint(getenv("PAIR_LAUNCH_ORDINAL"), 10, 64)
	if e == nil && ordinal != 0 && getenv("PAIR_LAUNCH_NONCE") != "" {
		e = PublishCaptureWitness(raw, events, artifactpath.CaptureWitness{ScopeKey: getenv("PAIR_SCOPE_KEY"), Tag: getenv("PAIR_TAG"), Agent: getenv("PAIR_AGENT"), LaunchOrdinal: ordinal, Nonce: getenv("PAIR_LAUNCH_NONCE")})
		if e != nil {
			raw.Close()
			events.Close()
			return nil, nil, e
		}
	}
	return raw, events, nil
}
