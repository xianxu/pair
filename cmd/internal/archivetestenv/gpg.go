// Package archivetestenv provides isolated real-tool fixtures without depending
// on the archive product package or a test harness.
package archivetestenv

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// GPGHome owns one disposable encryption key and the private home containing it.
// Close stops only this home's agent and removes only the directory it created.
// pair:m5-concept integration
type GPGHome struct {
	Home        string
	Fingerprint string
	ownedHome   string
	closeOnce   sync.Once
	closeErr    error
}

func privateCommand(ctx context.Context, home, binary string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = append(os.Environ(), "HOME="+home, "GNUPGHOME="+home, "XDG_CONFIG_HOME="+home)
	cmd.WaitDelay = time.Second
	return cmd
}

// NewGPGHome creates a short, canonical private home and an unpassphrased key
// used only by real-tool conformance fixtures. No operator keyring is loaded.
func NewGPGHome(ctx context.Context) (fixture *GPGHome, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for _, binary := range []string{"gpg", "gpgconf"} {
		if _, err := exec.LookPath(binary); err != nil {
			return nil, err
		}
	}
	home, err := os.MkdirTemp(os.TempDir(), "pgc-")
	if err != nil {
		return nil, err
	}
	canonical, err := filepath.EvalSymlinks(home)
	if err != nil {
		return nil, errors.Join(err, os.RemoveAll(home))
	}
	fixture = &GPGHome{Home: canonical, ownedHome: canonical}
	defer func(owned *GPGHome) {
		if err != nil {
			err = errors.Join(err, owned.Close())
		}
	}(fixture)
	if err = os.Chmod(canonical, 0700); err != nil {
		return nil, err
	}
	gen := privateCommand(ctx, canonical, "gpg", "--no-options", "--homedir", canonical, "--batch", "--pinentry-mode", "loopback", "--passphrase", "", "--quick-generate-key", "Pair archive test <archive@example.invalid>", "rsa2048", "encr", "1d")
	if output, genErr := gen.CombinedOutput(); genErr != nil {
		return nil, fmt.Errorf("generate disposable encryption key: %w (%s)", genErr, strings.TrimSpace(string(output)))
	}
	listing, listErr := privateCommand(ctx, canonical, "gpg", "--no-options", "--homedir", canonical, "--batch", "--with-colons", "--list-keys").Output()
	if listErr != nil {
		return nil, fmt.Errorf("list disposable public key: %w", listErr)
	}
	for _, line := range strings.Split(string(listing), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) <= 9 || fields[0] != "fpr" {
			continue
		}
		fingerprint := fields[9]
		if len(fingerprint) != 40 && len(fingerprint) != 64 {
			return nil, errors.New("invalid disposable key fingerprint length")
		}
		if _, err = hex.DecodeString(fingerprint); err != nil {
			return nil, err
		}
		fixture.Fingerprint = fingerprint
		break
	}
	if fixture.Fingerprint == "" {
		return nil, errors.New("disposable key has no fingerprint")
	}
	return fixture, nil
}

func (g *GPGHome) Close() error {
	if g == nil || g.ownedHome == "" {
		return nil
	}
	g.closeOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		output, killErr := privateCommand(ctx, g.ownedHome, "gpgconf", "--homedir", g.ownedHome, "--kill", "gpg-agent").CombinedOutput()
		if killErr != nil {
			killErr = fmt.Errorf("stop disposable key agent: %w (%s)", killErr, strings.TrimSpace(string(output)))
		}
		g.closeErr = errors.Join(killErr, os.RemoveAll(g.ownedHome))
	})
	return g.closeErr
}
