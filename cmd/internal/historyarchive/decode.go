package historyarchive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// Decoder transforms encrypted compressed bytes into a streaming tar archive.
// On failure or cancellation it may close input and output implementing
// io.Closer. Blocking input must be closable, and output writes must unblock
// when closed or on cancellation. Successful decoding leaves output open.
type Decoder interface {
	Decode(context.Context, io.Reader, io.Writer) error
}

// CommandDecoder streams GPG output directly into zstd without plaintext files
// or an intermediate plaintext buffer. Home optionally isolates GNUPGHOME;
// normal gpg-agent and pinentry remain available for unlocking a secret key.
// The caller owns the operation's timeout and both stream lifetimes.
type CommandDecoder struct{ GPG, Zstd, Home string }

func (c CommandDecoder) Decode(parent context.Context, in io.Reader, out io.Writer) (result error) {
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	watched := make(chan struct{})
	go func() {
		defer close(watched)
		select {
		case <-ctx.Done():
			if closer, ok := in.(io.Closer); ok {
				_ = closer.Close()
			}
			if closer, ok := out.(io.Closer); ok {
				_ = closer.Close()
			}
		case <-done:
		}
	}()
	defer func() {
		if result != nil {
			cancel()
			<-watched
			close(done)
		} else {
			close(done)
			<-watched
			cancel()
		}
		if parent.Err() != nil {
			result = errors.Join(result, parent.Err())
		}
	}()
	if err := parent.Err(); err != nil {
		return err
	}
	compressedRead, compressedWrite, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("decoder compressed pipe: %w", err)
	}
	defer compressedRead.Close()
	defer compressedWrite.Close()
	inputRead, inputWrite, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("decoder input pipe: %w", err)
	}
	defer inputRead.Close()
	defer inputWrite.Close()
	gpgBinary := c.GPG
	if gpgBinary == "" {
		gpgBinary = "gpg"
	}
	gpg := exec.CommandContext(ctx, gpgBinary, "--no-options", "--no-auto-key-retrieve", "--auto-key-locate", "clear", "--output", "-", "--decrypt")
	if c.Home != "" {
		gpg.Env = append(os.Environ(), "GNUPGHOME="+c.Home)
	}
	gpg.WaitDelay = time.Second
	gpg.Stdin = inputRead
	gpg.Stdout = compressedWrite
	zstdBinary := c.Zstd
	if zstdBinary == "" {
		zstdBinary = "zstd"
	}
	zstd := exec.CommandContext(ctx, zstdBinary, "--decompress", "--stdout", "--quiet")
	zstd.WaitDelay = time.Second
	zstd.Stdin = compressedRead
	zstd.Stdout = out
	gd := &codecLimitedBuffer{limit: codecDiagnosticLimit}
	zd := &codecLimitedBuffer{limit: codecDiagnosticLimit}
	gpg.Stderr = gd
	zstd.Stderr = zd
	if err = gpg.Start(); err != nil {
		cancel()
		return codecPhase("gpg decrypt start", err, gd)
	}
	if err = zstd.Start(); err != nil {
		cancel()
		compressedRead.Close()
		compressedWrite.Close()
		inputRead.Close()
		inputWrite.Close()
		waitErr := gpg.Wait()
		return errors.Join(codecPhase("zstd decompress start", err, zd), codecPhase("gpg decrypt wait", waitErr, gd))
	}
	compressedRead.Close()
	compressedWrite.Close()
	inputRead.Close()
	results := make(chan error, 3)
	go func() {
		_, err := io.Copy(inputWrite, in)
		closeErr := inputWrite.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			err = fmt.Errorf("decoder input: %w", err)
		}
		results <- err
	}()
	go func() { results <- codecPhase("gpg decrypt", gpg.Wait(), gd) }()
	go func() { results <- codecPhase("zstd decompress", zstd.Wait(), zd) }()
	for range 3 {
		if err := <-results; err != nil {
			result = errors.Join(result, err)
			cancel()
			inputWrite.Close()
		}
	}
	return result
}

var _ Decoder = CommandDecoder{}
