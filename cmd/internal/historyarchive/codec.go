package historyarchive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Codec transforms a tar stream into encrypted compressed bytes.
type Codec interface {
	CheckRecipients(context.Context, []string) error
	Encode(context.Context, io.Reader, io.Writer, []string) error
}

// CommandCodec uses public keys only. Home optionally isolates GNUPGHOME.
// On cancellation/failure Encode closes an input implementing io.Closer.
// Inputs that can block must implement io.Closer to interrupt Read; output
// writes must return (the archive coordinator uses a regular staging file).
type CommandCodec struct{ GPG, Zstd, Home string }

const codecDiagnosticLimit = 4096

type codecLimitedBuffer struct {
	data  []byte
	limit int
}

func (b *codecLimitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := b.limit - len(b.data)
	if remaining > len(p) {
		remaining = len(p)
	}
	if remaining > 0 {
		b.data = append(b.data, p[:remaining]...)
	}
	return n, nil
}
func (b *codecLimitedBuffer) String() string { return strings.TrimSpace(string(b.data)) }
func (c CommandCodec) gpg(ctx context.Context, args ...string) *exec.Cmd {
	binary := c.GPG
	if binary == "" {
		binary = "gpg"
	}
	base := []string{"--no-options", "--batch", "--no-tty", "--no-auto-key-retrieve", "--auto-key-locate", "clear", "--pinentry-mode", "error"}
	cmd := exec.CommandContext(ctx, binary, append(base, args...)...)
	if c.Home != "" {
		cmd.Env = append(os.Environ(), "GNUPGHOME="+c.Home)
	}
	cmd.WaitDelay = time.Second
	return cmd
}
func codecFingerprints(recipients []string) error {
	if len(recipients) == 0 {
		return errors.New("gpg recipients: at least one fingerprint required")
	}
	for _, fp := range recipients {
		if len(fp) != 40 && len(fp) != 64 {
			return errors.New("gpg recipients: full fingerprint required")
		}
		for _, r := range fp {
			if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
				return errors.New("gpg recipients: invalid fingerprint")
			}
		}
	}
	return nil
}
func codecPhase(phase string, err error, diag *codecLimitedBuffer) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w (%s)", phase, err, diag.String())
}

// CheckRecipients checks current wall-clock expiry and exact primary or subkey
// fingerprints from machine-readable public listings. Primary fingerprints may
// select a usable encryption subkey; aggregate capability flags alone do not
// establish that a nonexpired, nonrevoked encryption key is available.
func (c CommandCodec) CheckRecipients(ctx context.Context, recipients []string) error {
	if err := codecFingerprints(recipients); err != nil {
		return err
	}
	for _, fp := range recipients {
		cmd := c.gpg(ctx, "--with-colons", "--fixed-list-mode", "--with-fingerprint", "--with-subkey-fingerprint", "--list-keys", "--", fp)
		out := &codecLimitedBuffer{limit: 1 << 20}
		diag := &codecLimitedBuffer{limit: codecDiagnosticLimit}
		cmd.Stdout = out
		cmd.Stderr = diag
		if err := cmd.Run(); err != nil {
			return codecPhase("gpg recipient lookup", err, diag)
		}
		if len(out.data) == out.limit {
			return errors.New("gpg recipient listing exceeds limit")
		}
		now := time.Now().Unix()
		usable := false
		primaryOK := false
		primaryMatch := false
		kind := ""
		valid := false
		capability := ""
		for _, line := range strings.Split(string(out.data), "\n") {
			fields := strings.Split(line, ":")
			if len(fields) < 10 {
				continue
			}
			switch fields[0] {
			case "pub", "sub":
				kind = fields[0]
				valid = !strings.ContainsAny(fields[1], "redi")
				capability = ""
				if len(fields) > 11 {
					capability = fields[11]
					if strings.Contains(capability, "D") {
						valid = false
					}
				}
				if fields[6] != "" && fields[6] != "0" {
					expiry, e := strconv.ParseInt(fields[6], 10, 64)
					if e != nil || expiry <= now {
						valid = false
					}
				}
				if kind == "pub" {
					primaryOK = valid
					primaryMatch = false
				}
			case "fpr":
				if kind == "pub" {
					primaryMatch = strings.EqualFold(fields[9], fp)
				}
				if (strings.EqualFold(fields[9], fp) || primaryMatch) && valid && primaryOK && strings.Contains(capability, "e") {
					usable = true
				}
			}
		}
		if !usable {
			return fmt.Errorf("gpg recipient %s: missing, expired, revoked, or not encryption-capable", fp)
		}
	}
	return nil
}

func (c CommandCodec) Encode(ctx context.Context, in io.Reader, out io.Writer, recipients []string) error {
	if err := codecFingerprints(recipients); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan struct{})
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		select {
		case <-ctx.Done():
			if closer, ok := in.(io.Closer); ok {
				closer.Close()
			}
		case <-done:
		}
	}()
	defer func() { close(done); <-watchDone }()
	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("codec pipe: %w", err)
	}
	defer r.Close()
	defer w.Close()
	args := []string{"--trust-model", "always", "--compress-algo", "none", "--output", "-", "--encrypt"}
	for _, fp := range recipients {
		args = append(args, "--recipient", fp)
	}
	gpg := c.gpg(ctx, args...)
	gpg.Stdin = r
	gpg.Stdout = out
	binary := c.Zstd
	if binary == "" {
		binary = "zstd"
	}
	zstd := exec.CommandContext(ctx, binary, "-19", "--single-thread", "--stdout", "--quiet")
	zstd.WaitDelay = time.Second
	inputRead, inputWrite, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("codec input pipe: %w", err)
	}
	defer inputRead.Close()
	defer inputWrite.Close()
	zstd.Stdin = inputRead
	zstd.Stdout = w
	gd := &codecLimitedBuffer{limit: codecDiagnosticLimit}
	zd := &codecLimitedBuffer{limit: codecDiagnosticLimit}
	gpg.Stderr = gd
	zstd.Stderr = zd
	if err = gpg.Start(); err != nil {
		cancel()
		return codecPhase("gpg start", err, gd)
	}
	if err = zstd.Start(); err != nil {
		cancel()
		r.Close()
		w.Close()
		waitErr := gpg.Wait()
		return errors.Join(codecPhase("zstd start", err, zd), codecPhase("gpg wait", waitErr, gd))
	}
	r.Close()
	w.Close()
	inputRead.Close()
	results := make(chan error, 3)
	go func() {
		_, err := io.Copy(inputWrite, in)
		inputWrite.Close()
		if err != nil {
			err = fmt.Errorf("codec input: %w", err)
		}
		results <- err
	}()
	go func() { results <- codecPhase("zstd stream", zstd.Wait(), zd) }()
	go func() { results <- codecPhase("gpg stream", gpg.Wait(), gd) }()
	var result error
	for i := 0; i < 3; i++ {
		if e := <-results; e != nil {
			result = errors.Join(result, e)
			cancel()
			inputWrite.Close()
		}
	}
	if result == nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return result
}
