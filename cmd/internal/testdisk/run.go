package testdisk

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
)

type runOptions struct {
	name      string
	perRun    uint64
	repeats   uint64
	supported bool
	uncovered string
}

func runMeasured(options runOptions, read func() (uint64, error), run func() int, out io.Writer) int {
	if _, err := CheckBytes(0, 0, options.perRun, options.repeats); err != nil {
		fmt.Fprintf(out, "write budget %s: %v\n", options.name, err)
		return 1
	}
	if options.uncovered != "" {
		status := run()
		fmt.Fprintf(out, "write budget %s: %s workload outside ordinary test cap\n", options.name, options.uncovered)
		return status
	}
	if !options.supported {
		status := run()
		fmt.Fprintf(out, "write budget %s: unsupported; native self counter requires Darwin+cgo\n", options.name)
		return status
	}
	before, beforeErr := read()
	status := run()
	fail := func(stage string, err error) int {
		fmt.Fprintf(out, "write budget %s: %s%v\n", options.name, stage, err)
		if status != 0 {
			return status
		}
		return 1
	}
	if beforeErr != nil {
		return fail("before sample: ", beforeErr)
	}
	after, err := read()
	if err != nil {
		return fail("after sample: ", err)
	}
	verdict, err := CheckBytes(before, after, options.perRun, options.repeats)
	if err != nil {
		return fail("", err)
	}
	fmt.Fprintf(out, "write budget %s: bytes=%d cap=%d repeats=%d\n", options.name, verdict.Bytes, verdict.Limit, options.repeats)
	return status
}

// RunWithWriteBudget surrounds the existing ordinary TestMain callback. Helper
// dispatch must happen before it, and callback errors/panics keep their meaning.
func RunWithWriteBudget(name string, perRun uint64, run func() int) int {
	if !flag.Parsed() {
		flag.Parse()
	}
	value := func(name string) string {
		if f := flag.Lookup(name); f != nil {
			return f.Value.String()
		}
		return ""
	}
	count, err := strconv.Atoi(value("test.count"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "write budget %s: missing test count: %v\n", name, err)
		return 1
	}
	repeats, err := invocationRuns(count, value("test.cpu"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "write budget %s: %v\n", name, err)
		return 1
	}
	opts := runOptions{name: name, perRun: perRun, repeats: repeats, supported: nativeBudgetSupported}
	if value("test.bench") != "" {
		opts.uncovered = "benchmark"
	}
	if value("test.fuzz") != "" || value("test.fuzzworker") == "true" {
		opts.uncovered = "fuzz"
	}
	if opts.supported && opts.uncovered == "" {
		// Go's cache-access trace measures observation traffic, not fixture
		// writes. Omit it entirely: cmd/go refuses result-cache publication
		// without this log. An empty valid log would cache unobserved inputs.
		tracing := value("test.testlogfile") != ""
		if err := flag.Set("test.testlogfile", ""); err != nil {
			fmt.Fprintf(os.Stderr, "write budget %s: disable test-result cache: %v\n", name, err)
			return 1
		}
		if tracing {
			fmt.Fprintf(os.Stderr, "write budget %s: test-result cache disabled; build cache unchanged\n", name)
		}
	}
	return runMeasured(opts, selfWriteReader(), run, os.Stderr)
}
