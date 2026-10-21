// The harness places physical copies of this executable named claude/codex.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
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
	if size, e := pty.GetsizeFull(os.Stdin); e == nil {
		config.Width, config.Height = int(size.Cols), int(size.Rows)
	}
	previous, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	defer term.Restore(int(os.Stdin.Fd()), previous)
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
				if size, e := pty.GetsizeFull(os.Stdin); e == nil {
					next := agentfixture.Size{Cols: int(size.Cols), Rows: int(size.Rows)}
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
	if err := agentfixture.Run(ctx, config, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
