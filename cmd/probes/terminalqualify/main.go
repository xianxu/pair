// terminalqualify reports compatibility of the pinned candidate. A successful
// diagnostic run may exit 1: that means required terminal behavior is unmet.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	q "github.com/xianxu/pair/cmd/internal/terminalqualify"
	"io"
	"os"
	"time"
)

func run(ctx context.Context, out, diagnostic io.Writer, runner func(context.Context) (q.Report, error)) int {
	report, err := runner(ctx)
	if err != nil {
		fmt.Fprintln(diagnostic, "qualification infrastructure:", err)
		return 2
	}
	if err = report.Validate(); err != nil {
		fmt.Fprintln(diagnostic, "invalid qualification report:", err)
		return 2
	}
	doc := struct {
		q.Report
		Qualified bool   `json:"qualified"`
		Summary   string `json:"summary"`
	}{report, report.Qualified(), report.Summary()}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(doc); err != nil {
		fmt.Fprintln(diagnostic, "write report:", err)
		return 2
	}
	if !report.Qualified() {
		return 1
	}
	return 0
}
func main() {
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: terminalqualify (writes JSON; exit 1 means not qualified)")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	code := run(ctx, os.Stdout, os.Stderr, q.Run)
	cancel()
	os.Exit(code)
}
