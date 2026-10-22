// The development command starts ONLY the real application. Couch owns the front.
package main

import (
	"context"
	"example.com/mock-to-real/internal/app"
	"example.com/mock-to-real/internal/catalogue"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8081", "real HTTP address")
	flag.Parse()
	if e := run(*listen); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run(address string) error {
	f, e := os.Open("mocks/screens.json")
	if e != nil {
		return e
	}
	screens, e := catalogue.Read(f)
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	return app.Serve(ctx, address, screens)
}
