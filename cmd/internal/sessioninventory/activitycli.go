package sessioninventory

import (
	"encoding/json"
	"fmt"
	"io"
)

// pair:155-concept integration new final activity query
func runActivityCLI(options cliOptions, runtime Runtime, stdout, stderr io.Writer) int {
	query, err := QuerySession(runtime, options.currentScopeKey, options.currentTag, options.agents[0])
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "pair session-inventory: activity query failed")
		return 2
	}
	activity, ok, err := ActivityForSession(runtime, query)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "pair session-inventory: activity query failed")
		return 2
	}
	if !ok {
		return 0
	}
	rendered, err := json.Marshal(activity)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "pair session-inventory: activity render failed")
		return 2
	}
	rendered = append(rendered, '\n')
	if err := writeBuffered(stdout, rendered); err != nil {
		_, _ = fmt.Fprintln(stderr, "pair session-inventory: activity render write failed")
		return 2
	}
	return 0
}
