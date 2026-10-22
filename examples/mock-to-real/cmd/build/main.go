package main

import (
	"context"
	"example.com/mock-to-real/internal/buildsample"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	output := flag.String("output", "", "new production binary path (required)")
	flag.Parse()
	if *output == "" {
		fmt.Fprintln(os.Stderr, "--output PATH is required")
		os.Exit(2)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if e := buildsample.Build(ctx, ".", *output, os.Stdout); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
