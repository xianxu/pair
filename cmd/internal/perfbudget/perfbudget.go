// Package perfbudget is the one switch for wall-clock budgets in tests. A
// budget measures the machine as much as the code, so on a shared, loaded
// host it flakes (pair#436, pair#481): the default suite runs the scenario,
// asserts its outcome and logs the time; `make test-perf-budgets`
// (PAIR_PERF_BUDGETS=1) asserts the budget on a quiet host, in isolation.
//
// Prefer an operation count where one exists: load can't fake it.
package perfbudget

import (
	"os"
	"testing"
	"time"
)

// Env enables the budgets.
const Env = "PAIR_PERF_BUDGETS"

// Enforced says whether this run asserts budgets.
func Enforced() bool { return os.Getenv(Env) == "1" }

// Check logs what took elapsed, and fails t if it's over budget while
// budgets are enforced.
func Check(t testing.TB, what string, elapsed, budget time.Duration) {
	t.Helper()
	t.Logf("%s: %v (budget %v)", what, elapsed, budget)
	if elapsed > budget && Enforced() {
		t.Fatalf("%s took %v, over its %v budget", what, elapsed, budget)
	}
}
