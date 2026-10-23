//go:build darwin || linux

package vmcmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const maxPositionClaim = 64 << 10

// Position holds a stable kernel lock for exactly one lane. Closing it never
// removes the lock inode or interprets the retained launch claim.
type Position struct {
	mu        sync.Mutex
	dir       string
	lane      Lane
	directory *os.File
	lock      *os.File
}

func AcquirePosition(ctx context.Context, root string, lane Lane, progress io.Writer) (*Position, error) {
	if lane != TrainLane && lane != AdhocLane {
		return nil, fmt.Errorf("invalid VM position lane %q", lane)
	}
	if err := validateRunPath(root); err != nil || root == string(filepath.Separator) {
		return nil, fmt.Errorf("invalid VM position root %q", root)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Parents are owned by the caller; do not chmod or create them.
	if err := os.Mkdir(root, 0700); err != nil && !os.IsExist(err) {
		return nil, err
	}
	rootfd, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	defer unix.Close(rootfd)
	if err := privateDirectory(rootfd); err != nil {
		return nil, err
	}
	if err := unix.Mkdirat(rootfd, string(lane), 0700); err != nil && !errors.Is(err, unix.EEXIST) {
		return nil, err
	}
	fd, err := unix.Openat(rootfd, string(lane), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	directory := os.NewFile(uintptr(fd), filepath.Join(root, string(lane)))
	success := false
	defer func() {
		if !success {
			directory.Close()
		}
	}()
	if err := privateDirectory(fd); err != nil {
		return nil, err
	}
	lockfd, err := unix.Openat(fd, "position.lock", unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0600)
	if err != nil {
		return nil, err
	}
	lock := os.NewFile(uintptr(lockfd), "position.lock")
	defer func() {
		if !success {
			lock.Close()
		}
	}()
	if err := privateRegular(lockfd); err != nil {
		return nil, err
	}
	waiting := false
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		err = unix.Flock(lockfd, unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) && !errors.Is(err, unix.EINTR) {
			return nil, err
		}
		if !waiting {
			waiting = true
			if progress != nil {
				if _, err := fmt.Fprintf(progress, "Waiting for VM position %s\n", lane); err != nil {
					return nil, err
				}
			}
		}
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	success = true
	return &Position{dir: filepath.Join(root, string(lane)), lane: lane, directory: directory, lock: lock}, nil
}
func privateDirectory(fd int) error {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Uid != uint32(os.Getuid()) || st.Mode&07777 != 0700 {
		return fmt.Errorf("VM position directory must be private 0700 and owned by current user")
	}
	return nil
}
func privateRegular(fd int) error {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != uint32(os.Getuid()) || st.Mode&07777 != 0600 || st.Nlink != 1 {
		return fmt.Errorf("VM position file must be singly linked regular 0600 and owned by current user")
	}
	return nil
}
func (p *Position) Dir() string { return p.dir }
func (p *Position) ReadClaim() (LaunchClaim, bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.readClaim()
}
func (p *Position) readClaim() (LaunchClaim, bool, error) {
	if p.directory == nil {
		return LaunchClaim{}, false, os.ErrClosed
	}
	fd, err := unix.Openat(int(p.directory.Fd()), "claim.json", unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if errors.Is(err, unix.ENOENT) {
		return LaunchClaim{}, false, nil
	}
	if err != nil {
		return LaunchClaim{}, false, err
	}
	f := os.NewFile(uintptr(fd), "claim.json")
	defer f.Close()
	if err := privateRegular(fd); err != nil {
		return LaunchClaim{}, false, err
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxPositionClaim+1))
	if err != nil {
		return LaunchClaim{}, false, err
	}
	if len(raw) > maxPositionClaim {
		return LaunchClaim{}, false, fmt.Errorf("VM launch claim exceeds 64 KiB")
	}
	c, err := DecodeLaunchClaim(raw)
	if err != nil {
		return LaunchClaim{}, false, err
	}
	if c.Lane != p.lane {
		return LaunchClaim{}, false, fmt.Errorf("VM launch claim belongs to another lane")
	}
	return c, true, nil
}
func (p *Position) WriteClaim(c LaunchClaim) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if c.Lane != p.lane {
		return fmt.Errorf("VM launch claim belongs to another lane")
	}
	old, exists, err := p.readClaim()
	if err != nil {
		return err
	}
	if exists {
		if old.Nonce != c.Nonce || old.Lane != c.Lane || old.VM != c.VM {
			return fmt.Errorf("VM launch claim generation mismatch")
		}
		merged := make(map[string]json.RawMessage, len(old.Unknown)+len(c.Unknown))
		for k, v := range old.Unknown {
			merged[k] = v
		}
		for k, v := range c.Unknown {
			merged[k] = v
		}
		c.Unknown = merged
	}
	raw, err := EncodeLaunchClaim(c)
	if err != nil {
		return err
	}
	if len(raw) > maxPositionClaim {
		return fmt.Errorf("VM launch claim exceeds 64 KiB")
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	name := ".claim-" + hex.EncodeToString(nonce[:])
	dirfd := int(p.directory.Fd())
	fd, err := unix.Openat(dirfd, name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(fd), name)
	defer f.Close()
	defer unix.Unlinkat(dirfd, name, 0)
	if _, err = f.Write(raw); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = unix.Renameat(dirfd, name, dirfd, "claim.json"); err != nil {
		return err
	}
	return p.directory.Sync()
}
func (p *Position) ClearClaim(nonce string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	c, exists, err := p.readClaim()
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if c.Nonce != nonce {
		return fmt.Errorf("VM launch claim generation mismatch")
	}
	if err := unix.Unlinkat(int(p.directory.Fd()), "claim.json", 0); err != nil {
		return err
	}
	return p.directory.Sync()
}
func (p *Position) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.directory == nil {
		return nil
	}
	err := p.lock.Close()
	err = errors.Join(err, p.directory.Close())
	p.lock = nil
	p.directory = nil
	return err
}
