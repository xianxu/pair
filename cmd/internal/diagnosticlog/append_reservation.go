package diagnosticlog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const maxReservationBytes = 4096

type appendReservation struct {
	Version  int        `json:"version"`
	Before   generation `json:"before"`
	Length   int        `json:"length"`
	Digest   string     `json:"digest"`
	At       time.Time  `json:"at"`
	Checksum string     `json:"checksum"`
}

func reservationChecksum(r appendReservation) string {
	r.Checksum = ""
	b, _ := json.Marshal(r)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func ValidateAppendReservation(r appendReservation) error {
	if r.Before.UnverifiedTail != nil || r.Version != 1 || r.Length <= 0 || r.Length > maxAppendBytes || r.Before.Size < 0 || r.Before.Size > math.MaxInt64-int64(r.Length) || r.Before.Name != "" || r.Before.Identity == (identity{}) || r.Before.Start.IsZero() || r.Before.LastWrite.IsZero() || r.Before.ModTime.IsZero() || r.At.IsZero() {
		return errors.New("invalid diagnostic append reservation")
	}
	digest, e := hex.DecodeString(r.Digest)
	if e != nil || len(digest) != sha256.Size || r.Checksum != reservationChecksum(r) {
		return errors.New("invalid diagnostic reservation digest")
	}
	return nil
}

func DecodeAppendReservation(b []byte) (appendReservation, error) {
	var r appendReservation
	if len(b) > maxReservationBytes {
		return r, errors.New("oversized diagnostic reservation")
	}
	if e := strictjson.Decode(b, &r); e != nil {
		return r, e
	}
	return r, ValidateAppendReservation(r)
}

func sameGeneration(a, b generation) bool {
	return a.Name == b.Name && a.Identity == b.Identity && a.Size == b.Size && a.Start.Equal(b.Start) && a.LastWrite.Equal(b.LastWrite) && a.ModTime.Equal(b.ModTime)
}

// ClassifyAppendRecovery certifies a complete reserved batch. An arbitrary
// killed partial syscall has no full digest: retain it as unknown evidence.
func ClassifyAppendRecovery(current, observed generation, tail []byte, r appendReservation) (generation, error) {
	if e := ValidateAppendReservation(r); e != nil {
		return generation{}, e
	}
	if !sameGeneration(current, r.Before) || observed.Identity != current.Identity || observed.Size-current.Size != int64(r.Length) || len(tail) != r.Length {
		return generation{}, errors.New("diagnostic reservation identity or size changed")
	}
	sum := sha256.Sum256(tail)
	if hex.EncodeToString(sum[:]) != r.Digest {
		return generation{}, errors.New("diagnostic reserved tail changed")
	}
	next := current
	next.Size = observed.Size
	next.ModTime = observed.ModTime
	next.LastWrite = r.At
	return next, nil
}

func reservationPath(path string) string {
	return filepath.Join(directory(path), ".append-reservation")
}

func writeReservation(path string, before generation, data []byte, at time.Time, o Options) error {
	sum := sha256.Sum256(data)
	r := appendReservation{Version: 1, Before: before, Length: len(data), Digest: hex.EncodeToString(sum[:]), At: at}
	r.Checksum = reservationChecksum(r)
	if e := ValidateAppendReservation(r); e != nil {
		return e
	}
	b, e := json.Marshal(r)
	if e != nil {
		return e
	}
	if len(b) > maxReservationBytes {
		return errors.New("oversized diagnostic reservation")
	}
	if e = o.checkContext(); e != nil {
		return e
	}
	f, e := openRegular(reservationPath(path), syscall.O_RDWR|syscall.O_CREAT)
	if e != nil {
		return e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return e
	}
	if st.Size() > maxReservationBytes || st.Sys().(*syscall.Stat_t).Nlink != 1 {
		return errors.New("unsafe diagnostic reservation file")
	}
	if e = o.checkContext(); e != nil {
		return e
	}
	n, e := f.WriteAt(b, 0)
	o.observe("write", reservationPath(path), n)
	if e != nil {
		return e
	}
	if n != len(b) {
		return io.ErrShortWrite
	}
	// Payload effects start only after the bounded header is wholly published.
	// A process killed mid-header leaves the old payload checkpoint untouched.
	return f.Truncate(int64(len(b)))
}

func readReservation(path string) (appendReservation, error) {
	f, e := openRegular(reservationPath(path), syscall.O_RDONLY)
	if e != nil {
		return appendReservation{}, e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return appendReservation{}, e
	}
	if st.Sys().(*syscall.Stat_t).Nlink != 1 {
		return appendReservation{}, errors.New("unsafe diagnostic reservation file")
	}
	b, e := io.ReadAll(io.LimitReader(f, maxReservationBytes+1))
	if e != nil {
		return appendReservation{}, e
	}
	return DecodeAppendReservation(b)
}

func recoverReservedAppend(path string, s *diskState, o Options, durable bool) error {
	// These durable transactions own paths that can legitimately be absent or
	// renamed. Their recovery, not an obsolete append reservation, validates them.
	if s.Pending != nil || s.Deleting != nil || s.Creating != nil || s.Appending != nil {
		return nil
	}
	if s.Current.Identity == (identity{}) {
		return nil
	}
	st, e := regular(path)
	if e != nil {
		return e
	}
	match, checkpointErr := compareCheckpoint(observedGeneration(st), s.Current)
	if checkpointErr == nil {
		if match == checkpointDeviceOnly {
			next := *s
			next.Current.Identity = fileIdentity(st)
			if err := save(path, next, true, o); err != nil {
				return fmt.Errorf("diagnostic device rebind checkpoint: %w", err)
			}
			*s = next
		}
		return nil
	} // no uncheckpointed effect; stale/torn headers confer no authority
	r, e := readReservation(path)
	if e != nil {
		return errors.Join(checkpointErr, e)
	}
	if e = o.checkContext(); e != nil {
		return e
	}
	if sameGeneration(r.Before, s.Current) && fileIdentity(st) == s.Current.Identity && st.Size() > s.Current.Size && st.Size()-s.Current.Size < int64(r.Length) {
		observed := s.Current
		observed.Size = st.Size()
		observed.ModTime = st.ModTime()
		quarantined, e := ClassifyPartialQuarantine(s.Current, observed, r)
		if e != nil {
			return e
		}
		if e = o.prove(path, s.Writers); e != nil {
			return e
		}
		if e = rotateGeneration(path, s, quarantined, o); e != nil {
			return e
		}
		return ensureCurrent(path, s, o)
	}
	if !sameGeneration(r.Before, s.Current) || fileIdentity(st) != s.Current.Identity || st.Size()-s.Current.Size != int64(r.Length) {
		return errors.Join(errors.New("diagnostic reservation cannot recover checkpoint"), checkpointErr)
	}
	f, e := openRegular(path, syscall.O_RDWR)
	if e != nil {
		return e
	}
	defer f.Close()
	st, e = f.Stat()
	if e != nil {
		return e
	}
	tail := make([]byte, r.Length)
	if _, e = io.ReadFull(io.NewSectionReader(f, s.Current.Size, int64(r.Length)), tail); e != nil {
		return e
	}
	observed := s.Current
	observed.Size = st.Size()
	observed.ModTime = st.ModTime()
	observed.Identity = fileIdentity(st)
	next, e := ClassifyAppendRecovery(s.Current, observed, tail, r)
	if e != nil {
		return e
	}
	verify, e := f.Stat()
	if e != nil {
		return e
	}
	if e = matches(verify, next); e != nil {
		return e
	}
	if e = o.checkContext(); e != nil {
		return e
	}
	if durable {
		if e = syncAppendPayload(f, o); e != nil {
			return e
		}
	}
	if e = fault(o, "append-verified"); e != nil {
		return e
	}
	checkpoint := *s
	checkpoint.Current = next
	if e = save(path, checkpoint, durable, o); e != nil {
		return e
	}
	*s = checkpoint
	return fault(o, "append-committed")
}

// A reservation is reusable metadata, never an independent retained payload.
func removeReservation(path string, o Options) error {
	st, e := regular(reservationPath(path))
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	if st.Size() > maxReservationBytes || st.Sys().(*syscall.Stat_t).Nlink != 1 {
		return errors.New("unsafe diagnostic reservation retirement")
	}
	if e = o.checkContext(); e != nil {
		return e
	}
	if e = os.Remove(reservationPath(path)); e != nil {
		return e
	}
	return syncDir(directory(path))
}

func validUnverifiedTail(g generation) bool {
	q := g.UnverifiedTail
	if q == nil {
		return true
	}
	digest, e := hex.DecodeString(q.Digest)
	return e == nil && len(digest) == sha256.Size && q.Offset >= 0 && q.Length > 0 && q.Length <= maxAppendBytes && g.Size > q.Offset && g.Size-q.Offset < int64(q.Length)
}

// ClassifyPartialQuarantine labels observed incomplete bytes, never certifies
// their contents. Only a later writer-proved rotation may preserve this inode.
func ClassifyPartialQuarantine(current, observed generation, r appendReservation) (generation, error) {
	if e := ValidateAppendReservation(r); e != nil {
		return generation{}, e
	}
	if !sameGeneration(current, r.Before) || observed.Identity != current.Identity || observed.Size <= current.Size || observed.Size-current.Size >= int64(r.Length) || observed.ModTime.IsZero() {
		return generation{}, errors.New("not an incomplete reserved diagnostic batch")
	}
	g := current
	g.Size = observed.Size
	g.ModTime = observed.ModTime
	g.LastWrite = r.At
	g.UnverifiedTail = &unverifiedTail{Offset: current.Size, Length: r.Length, Digest: r.Digest}
	return g, nil
}

// Serialized uncertainty is compared by value: distinct JSON decodes allocate
// distinct pointers for the same exact evidence.
func generationMetadataEqual(a, b generation) bool {
	if !sameGeneration(a, b) {
		return false
	}
	if a.UnverifiedTail == nil || b.UnverifiedTail == nil {
		return a.UnverifiedTail == nil && b.UnverifiedTail == nil
	}
	return *a.UnverifiedTail == *b.UnverifiedTail
}
