// Package testbinary owns compiled test fixtures until their suite has finished.
// Production code must not use this test-support package.
package testbinary

import (
	"errors"
	"os"
	"sync"
)

// Cache shares one build attempt, including its failure, across test consumers.
// Its owner calls Close after m.Run, after every consumer's cleanup has finished.
// The zero value is ready to use.
type Cache struct {
	mu        sync.Mutex
	attempted bool
	closed    bool
	dir       string
	err       error
}

// Directory calls build once with an owned temporary directory. The callback
// may populate several executable artifacts that form one fixture.
func (c *Cache) Directory(build func(string) error) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return "", errors.New("test binary cache is closed")
	}
	if !c.attempted {
		c.attempted = true
		c.dir, c.err = os.MkdirTemp("", "pair-test-binaries-")
		if c.err == nil {
			c.err = build(c.dir)
		}
	}
	return c.dir, c.err
}

// Close joins concurrent lookups and removes every artifact, including partial
// output from a failed build. It must run only after all child consumers join.
func (c *Cache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	if c.dir == "" {
		return nil
	}
	err := os.RemoveAll(c.dir)
	if err == nil {
		c.dir = ""
	}
	return err
}
