// The harness places physical copies of this executable named claude/codex.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func main() { os.Exit(run()) }
func run() int {
	agent := filepath.Base(os.Args[0])
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		version := agentfixture.Version(agent)
		if version == "" {
			fmt.Fprintln(os.Stderr, "fixture executable must be named claude or codex")
			return 2
		}
		fmt.Println(version)
		return 0
	}
	config, err := agentfixture.ConfigFromEnvironment(agent, os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	// Resolve the descriptor before ttyio owns O_NONBLOCK. Calling os.File.Fd
	// again during resize can change pollability on the shared TTY description.
	inputFD := int(os.Stdin.Fd())
	if size, e := unix.IoctlGetWinsize(inputFD, unix.TIOCGWINSZ); e == nil {
		config.Width, config.Height = int(size.Col), int(size.Row)
	}
	previous, err := term.MakeRaw(inputFD)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	defer term.Restore(inputFD, previous)
	// Stdio belongs to this process, not the adapter: Close cancels and joins
	// transport operations and restores flags while leaving the borrowed FDs open.
	terminalIO, err := ttyio.NewFile(os.Stdin, os.Stdout, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	defer terminalIO.Close()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()
	winches := make(chan os.Signal, 1)
	signal.Notify(winches, syscall.SIGWINCH)
	defer signal.Stop(winches)
	sizes := make(chan agentfixture.Size, 1)
	config.Resizes = sizes
	resized := make(chan struct{})
	go func() {
		defer close(resized)
		for {
			select {
			case <-ctx.Done():
				return
			case <-winches:
				if size, e := unix.IoctlGetWinsize(inputFD, unix.TIOCGWINSZ); e == nil {
					next := agentfixture.Size{Cols: int(size.Col), Rows: int(size.Row)}
					select {
					case sizes <- next:
					default:
						select {
						case <-sizes:
						default:
						}
						select {
						case sizes <- next:
						default:
						}
					}
				}
			}
		}
	}()
	defer func() { cancel(); <-resized }()
	if err := agentfixture.Run(ctx, config, terminalIO, terminalOutput{ctx: ctx, file: terminalIO}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

// Output uses the process lifetime context so a stalled native terminal cannot
// prevent cancellation from reaching Run's reader/worker joins.
type terminalOutput struct {
	ctx  context.Context
	file *ttyio.File
}

func (w terminalOutput) Write(p []byte) (int, error) { return w.file.WriteContext(w.ctx, p) }
