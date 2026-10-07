package couchcore

import (
	"context"
	"fmt"
)

// OperationProgress receives one line describing what a long operation is
// doing now. recover runs several slow steps (#399 live acceptance: a reap
// alone waits out a SIGTERM the wrapper ignores), and the switcher shows each
// as it starts rather than one static "recovering…" line.
type OperationProgress func(text string)

type progressKey struct{}

// WithOperationProgress returns ctx carrying report.
func WithOperationProgress(ctx context.Context, report OperationProgress) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, progressKey{}, report)
}

// withProgressPrefix nests a reporter: every line reported under the returned
// context is prefixed, so a step's own phases read as part of that step.
func withProgressPrefix(ctx context.Context, prefix string) context.Context {
	parent, _ := ctx.Value(progressKey{}).(OperationProgress)
	if parent == nil {
		return ctx
	}
	return WithOperationProgress(ctx, func(text string) { parent(prefix + text) })
}

// reportProgress is a no-op without a reporter.
func reportProgress(ctx context.Context, format string, args ...any) {
	if ctx == nil {
		return
	}
	if report, _ := ctx.Value(progressKey{}).(OperationProgress); report != nil {
		report(fmt.Sprintf(format, args...))
	}
}
