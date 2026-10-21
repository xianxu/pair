package wrapcmd

import (
	"io"
	"os"
	"sync"
)

// startedPTY transfers the optional Darwin slave lease to the output owner.
// Only ownedGroup may reap the child; releasing this lease never waits.
type startedPTY struct {
	master     *os.File
	slave      io.Closer
	release    sync.Once
	releaseErr error
}

func (p *startedPTY) releaseSlave() error {
	p.release.Do(func() {
		if p.slave != nil {
			p.releaseErr = p.slave.Close()
		}
	})
	return p.releaseErr
}
