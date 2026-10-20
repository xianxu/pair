package couchcmd

import (
	"fmt"
	"io"

	"github.com/xianxu/pair/cmd/internal/webcast/query"
)

// verbCost is a public verb's relative cost on a loaded developer machine
// (pair#490): agents choose verbs without knowing a relaunch cold-starts a
// wrapper and an agent. The tiers come from timings recorded in the issue's
// Log; `couch --help` renders them and the shipped skill lists them, and a test
// keeps both equal to this table.
type verbCost string

const (
	costLight   verbCost = "light"   // under a second; reads Couch's memory or one small file
	costSeconds verbCost = "seconds" // a few seconds; reads git, recordings or transcripts
	costHeavy   verbCost = "heavy"   // tens of seconds, CPU-bound; starts or stops agents, or runs sdlc fleet-wide
)

// launchVerb names the bare `couch [path]` launch in verbCosts.
const launchVerb = "couch"

var verbCosts = map[string]verbCost{
	launchVerb:                 costHeavy,
	"--help":                   costLight,
	"--skill":                  costLight,
	"--actors":                 costLight,
	"--send-to":                costLight,
	"--move-slot":              costSeconds,
	"--notify":                 costLight,
	"--message-status":         costLight,
	"--operation-status":       costLight,
	"--drain":                  costLight,
	"--switch-agent":           costHeavy,
	"--keys":                   costSeconds,
	"--set-model":              costSeconds,
	"--interrupt":              costSeconds,
	"--peek":                   costLight,
	"--resources":              costLight,
	"--broadcast-list":         costLight,
	"--session-sync":           costLight,
	"--archived":               costLight,
	"--list":                   costSeconds,
	"--show":                   costSeconds,
	"--reconcile":              costSeconds,
	"--reap":                   costSeconds,
	"--reap-orphans":           costSeconds,
	"--adopt-store":            costLight,
	query.Flag:                 costLight,
	submissionFlag:             costLight,
	replayFlag:                 costLight,
	"--recover-plan-from-sdlc": costHeavy,
	"--resume":                 costHeavy,
	"--resume-candidates":      costSeconds,
	"--recovery-confirmation":  costHeavy,
	"--reboot":                 costHeavy,
	"--relaunch":               costHeavy,
	"--reload-context":         costHeavy,
	"--prepare":                costHeavy,
	"--recover":                costHeavy,
	"--recover-park":           costHeavy,
	fidelityFlag:               costSeconds,
}

// costGuidance is the one sentence both surfaces carry with the tags.
const costGuidance = "Cost: light is under a second, seconds is a few seconds, heavy is tens of\n" +
	"seconds of CPU (it stops or boots agents, or runs sdlc across the fleet);\n" +
	"run heavy verbs one at a time, after checking uptime's load."

// verbLine prints one usage line ending in its verb's cost tag. A verb missing
// from verbCosts panics, so a new verb cannot reach --help untagged.
func verbLine(w io.Writer, verb, line string) {
	cost, ok := verbCosts[verb]
	if !ok {
		panic("couch: no cost recorded for " + verb)
	}
	fmt.Fprintf(w, "%s  [%s]\n", line, cost)
}
