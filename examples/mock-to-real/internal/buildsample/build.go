// Package buildsample produces a real-only artifact in an isolated build stage.
// It is development tooling and is never a dependency of the resulting binary.
package buildsample

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"example.com/mock-to-real/internal/catalogue"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const productionMain = `package main
import("context";"encoding/json";"flag";"fmt";"os";"os/signal";"syscall";"example.com/mock-to-real/internal/app")
func main(){listen:=flag.String("listen","127.0.0.1:8081","HTTP address");check:=flag.Bool("check",false,"validate application configuration and exit");flag.Parse();var screens []app.Screen;if e:=json.Unmarshal([]byte(PROJECTION),&screens);e!=nil{panic(e)};if _,e:=app.New(screens);e!=nil{fmt.Fprintln(os.Stderr,e);os.Exit(1)};if *check{return};ctx,cancel:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer cancel();if e:=app.Serve(ctx,*listen,screens);e!=nil{fmt.Fprintln(os.Stderr,e);os.Exit(1)}}
`

func Build(ctx context.Context, root, output string, report io.Writer) (err error) {
	if e := ctx.Err(); e != nil {
		return e
	}
	if _, e := os.Lstat(output); e == nil {
		return fmt.Errorf("output already exists: %s", output)
	} else if !os.IsNotExist(e) {
		return e
	}
	file, e := os.Open(filepath.Join(root, "mocks/screens.json"))
	if e != nil {
		return e
	}
	screens, e := catalogue.Read(file)
	closeErr := file.Close()
	if e != nil {
		return e
	}
	if closeErr != nil {
		return closeErr
	}
	projection, e := json.Marshal(screens)
	if e != nil {
		return e
	}
	stage, e := os.MkdirTemp("", "mock-to-real-build-*")
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, os.RemoveAll(stage)) }()
	if e := os.MkdirAll(filepath.Join(stage, "internal/app"), 0755); e != nil {
		return e
	}
	if e := os.MkdirAll(filepath.Join(stage, "cmd/production"), 0755); e != nil {
		return e
	}
	module, e := os.ReadFile(filepath.Join(root, "go.mod"))
	if e != nil {
		return e
	}
	if e := os.WriteFile(filepath.Join(stage, "go.mod"), module, 0644); e != nil {
		return e
	}
	entries, e := os.ReadDir(filepath.Join(root, "internal/app"))
	if e != nil {
		return e
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		b, e := os.ReadFile(filepath.Join(root, "internal/app", entry.Name()))
		if e != nil {
			return e
		}
		if e := os.WriteFile(filepath.Join(stage, "internal/app", entry.Name()), b, 0644); e != nil {
			return e
		}
	}
	main := strings.Replace(productionMain, "PROJECTION", fmt.Sprintf("%q", projection), 1)
	if e := os.WriteFile(filepath.Join(stage, "cmd/production/main.go"), []byte(main), 0644); e != nil {
		return e
	}
	candidate := filepath.Join(stage, "application")
	if e := run(ctx, stage, "go", "build", "-trimpath", "-o", candidate, "./cmd/production"); e != nil {
		return e
	}
	if e := run(ctx, stage, candidate, "--check"); e != nil {
		return e
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	// O_EXCL makes existing-output behavior explicit, including publication races.
	out, e := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0755)
	if e != nil {
		return e
	}
	published := false
	defer func() {
		if !published {
			err = errors.Join(err, os.Remove(output))
		}
	}()
	in, e := os.Open(candidate)
	if e != nil {
		_ = out.Close()
		return e
	}
	_, copyErr := io.Copy(out, in)
	inErr := in.Close()
	outErr := out.Close()
	if e := errors.Join(copyErr, errors.Join(inErr, outErr)); e != nil {
		return e
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	for _, screen := range screens {
		if screen.State != "real" {
			if _, e := fmt.Fprintf(report, "Unimplemented: %s (%s)\n", screen.ID, screen.Route); e != nil {
				return e
			}
		}
	}
	_, e = fmt.Fprintf(report, "Built %s\n", output)
	if e == nil {
		published = true
	}
	return e
}

// Every child belongs to the build context; cancellation kills its process group,
// Run waits for exit, and WaitDelay bounds inherited pipe drainage (Unix sample).
func run(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		e := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if e == syscall.ESRCH {
			return os.ErrProcessDone
		}
		return e
	}
	cmd.WaitDelay = 5 * time.Second
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if e := cmd.Run(); e != nil {
		return fmt.Errorf("%s: %w: %s", name, e, &output)
	}
	return nil
}
