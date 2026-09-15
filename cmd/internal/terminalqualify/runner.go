package terminalqualify

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"
)

type Executor func(context.Context, Case, []string) (Observation, error)

func RunCase(ctx context.Context, c Case, execute Executor) (Result, error) {
	result := Result{ID: c.ID, Capability: c.Capability, Source: c.Source}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if c.Uncovered != "" {
		result.Status = NotCovered
		result.Detail = boundedDetail(c.Uncovered)
		return result, nil
	}
	if c.ID == "" || len(c.Expected) == 0 {
		return result, fmt.Errorf("case %q has no literal expectation", c.ID)
	}
	// Bound fixture partition metadata before constructing split variants.
	if len(c.Input) > 64*1024 {
		return result, fmt.Errorf("case %q exceeds 64KiB fixture limit", c.ID)
	}
	variants := [][]string{{c.Input}}
	if c.Split {
		variants = SplitInputs(c.Input)
	}
	for i, chunks := range variants {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		opctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		got, err := execute(opctx, c, chunks)
		cancel()
		if err != nil {
			return result, fmt.Errorf("%s variant %d: %w", c.ID, i, err)
		}
		if detail := Compare(c.Expected, got); detail != "" {
			result.Status = Fail
			result.Detail = mismatchDetails(fmt.Sprintf("variant=%d chunks=%d", i, len(chunks)), detail)
			return result, nil
		}
	}
	result.Status = Pass
	result.Detail = fmt.Sprintf("%d delivery variants matched literal expectations", len(variants))
	return result, nil
}
func CandidateVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/charmbracelet/x/vt" {
				v := dep.Path + "@" + dep.Version
				if dep.Replace != nil {
					v += " => " + dep.Replace.Path + "@" + dep.Replace.Version
				}
				return v
			}
		}
	}
	return "github.com/charmbracelet/x/vt (build version unavailable)"
}
func Cases() []Case {
	all := ScreenCases()
	all = append(all, InputCases()...)
	return append(all, Coverage()...)
}
func Run(ctx context.Context) (Report, error) {
	return RunMatrix(ctx, CandidateVersion(), Cases(), executeCandidate)
}
func RunMatrix(ctx context.Context, version string, cases []Case, execute Executor) (Report, error) {
	report := Report{Candidate: version}
	ids := map[string]bool{}
	for _, c := range cases {
		if c.ID == "" || ids[c.ID] {
			return report, fmt.Errorf("empty or duplicate case ID %q", c.ID)
		}
		ids[c.ID] = true
		report.Required = append(report.Required, c.ID)
	}
	for _, c := range cases {
		r, err := RunCase(ctx, c, execute)
		if err != nil {
			return report, err
		}
		report.Results = append(report.Results, r)
	}
	return report, report.Validate()
}
func executeCandidate(ctx context.Context, c Case, chunks []string) (Observation, error) {
	width, height := c.Width, c.Height
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}
	candidate, err := NewCandidate(width, height)
	if err != nil {
		return nil, err
	}
	defer candidate.Close()
	return candidate.Execute(ctx, chunks, c.Action)
}
