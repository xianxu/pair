package titlepoller

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type LegacyCutoff struct {
	Platform string `json:"platform"`
	BootID   string `json:"boot_id,omitempty"`
	Start    string `json:"start"`
}
type LegacyPolicy struct {
	Executable string
	DataRoot   string
	Cutoff     LegacyCutoff
}
type LegacyResource struct {
	RSSBytes uint64
	RSSKnown bool
}
type LegacySample struct {
	Rows      []procutil.Observation
	Complete  bool
	At        time.Time
	Platform  string
	BootID    string
	Resources map[int]LegacyResource
}
type LegacyDisposition string

const (
	LegacyKeep     LegacyDisposition = "keep"
	LegacyUnknown  LegacyDisposition = "unknown"
	LegacyEligible LegacyDisposition = "eligible"
)

type LegacyTitleDecision struct {
	PID         int               `json:"pid"`
	PPID        int               `json:"ppid"`
	Start       string            `json:"start"`
	Tag         string            `json:"tag,omitempty"`
	Agent       string            `json:"agent,omitempty"`
	Disposition LegacyDisposition `json:"disposition"`
	Reason      string            `json:"reason"`
	RSSBytes    uint64            `json:"rss_bytes"`
	RSSKnown    bool              `json:"rss_known"`
	AgeSeconds  float64           `json:"age_seconds"`
	AgeKnown    bool              `json:"age_known"`
}
type LegacyCleanupIO interface {
	Identity(int) (string, error)
	Signal(int, syscall.Signal) error
	Sleep(context.Context, time.Duration) error
}
type LegacyCleanupResult struct {
	PID   int
	State string
	Err   error
}

// DecideLegacyTitles grants no effects. One complete native process sample
// establishes wrapper absence; every uncertain possible wrapper keeps titles.
// Wrapper TAG comes from the existing exact artifact owner decoder, because
// production argv carries it in --scrollback-log rather than a separate flag.
func DecideLegacyTitles(sample LegacySample, policy LegacyPolicy) []LegacyTitleDecision {
	agents := make([]string, 0)
	for _, agent := range sessioninventory.SupportedAgents() {
		agents = append(agents, string(agent))
	}
	validPolicy := filepath.IsAbs(policy.Executable) && filepath.Clean(policy.Executable) == policy.Executable && filepath.Base(policy.Executable) == "pair" && filepath.IsAbs(policy.DataRoot) && filepath.Clean(policy.DataRoot) == policy.DataRoot
	cutoff, cutoffOK := legacyStart(policy.Cutoff.Platform, policy.Cutoff.Start)
	cutoffOK = cutoffOK && policy.Cutoff.Platform == sample.Platform && (sample.Platform != "linux" || (sample.BootID != "" && sample.BootID == policy.Cutoff.BootID))
	if sample.At.IsZero() {
		cutoffOK = false
	}
	if sample.Platform == "darwin" && cutoffOK {
		seconds := sample.At.Unix()
		micros := uint64(sample.At.Nanosecond() / 1000)
		cutoffOK = seconds >= 0 && (cutoff[0] < uint64(seconds) || (cutoff[0] == uint64(seconds) && cutoff[1] <= micros))
	}
	wrappers := map[string]bool{}
	uncertain := !sample.Complete || !validPolicy || !cutoffOK
	seen := map[int]bool{}
	for _, row := range sample.Rows {
		if seen[row.PID] {
			uncertain = true
		}
		seen[row.PID] = true
		if row.Zombie {
			continue
		}
		if row.Err != nil {
			if row.Name == "" || row.Name == "pair" || row.Name == "pair-wrap" {
				uncertain = true
			}
			continue
		}
		base := filepath.Base(row.Executable)
		var args []string
		switch {
		case base == "pair" && len(row.Args) > 1 && row.Args[1] == "wrap":
			args = row.Args[2:]
		case base == "pair-wrap":
			args = row.Args[1:]
		default:
			continue
		}
		if row.PID <= 1 || row.Identity == "" || len(row.Args) == 0 {
			uncertain = true
			continue
		}
		if _, ok := legacyStart(sample.Platform, row.Identity); !ok {
			uncertain = true
			continue
		}
		capture := ""
		malformed := false
		for len(args) > 0 && strings.HasPrefix(args[0], "-") {
			switch args[0] {
			case "--scrollback-log":
				if len(args) < 2 || capture != "" {
					malformed = true
					args = nil
					continue
				}
				capture = args[1]
				args = args[2:]
			case "--from-launch-env":
				args = args[1:]
			case "--":
				args = nil
			default:
				malformed = true
				args = nil
			}
		}
		owner, ok := artifactpath.SessionArtifactOwner(policy.DataRoot, capture, agents)
		raw := false
		if ok {
			paths, err := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
			if err == nil {
				for _, agent := range agents {
					artifacts, artifactErr := paths.ScrollbackArtifacts(agent)
					raw = raw || (artifactErr == nil && artifacts.Raw == capture)
				}
			}
		}
		if malformed || !ok || !raw {
			uncertain = true
			continue
		}
		wrappers[owner.Tag] = true
	}
	var out []LegacyTitleDecision
	for _, row := range sample.Rows {
		if row.Err != nil || row.Executable != policy.Executable || len(row.Args) != 5 || row.Args[0] != policy.Executable || row.Args[1] != "title" {
			continue
		}
		tag, agent := row.Args[2], row.Args[3]
		validAgent := false
		for _, supported := range agents {
			validAgent = validAgent || agent == supported
		}
		if _, err := artifactpath.ResolveScoped(policy.DataRoot, tag); err != nil || !validAgent || row.Args[4] == "" {
			continue
		}
		result := LegacyTitleDecision{PID: row.PID, PPID: row.PPID, Start: row.Identity, Tag: tag, Agent: agent, Disposition: LegacyUnknown, Reason: "incomplete or uncertain process sample"}
		if resource, ok := sample.Resources[row.PID]; ok {
			result.RSSBytes, result.RSSKnown = resource.RSSBytes, resource.RSSKnown
		}
		start, validStart := legacyStart(sample.Platform, row.Identity)
		if validStart && sample.Platform == "darwin" && !sample.At.IsZero() && sample.At.Unix() >= 0 && start[0] <= uint64(sample.At.Unix()) {
			birth := time.Unix(int64(start[0]), int64(start[1])*1000)
			if !birth.After(sample.At) {
				result.AgeSeconds = sample.At.Sub(birth).Seconds()
				result.AgeKnown = true
			}
		}
		switch {
		case row.PID <= 1 || row.PID > 1<<31-1 || !validStart:
			result.Reason = "unavailable process generation"
		case row.Zombie:
			result.Disposition = LegacyKeep
			result.Reason = "terminated row"
		case row.PPID != 1:
			result.Disposition = LegacyKeep
			result.Reason = "parent is not init"
		case !uncertain && !(start[0] < cutoff[0] || (start[0] == cutoff[0] && start[1] < cutoff[1])):
			result.Disposition = LegacyKeep
			result.Reason = "not older than disclosed lease activation cutoff"
		case wrappers[tag]:
			result.Disposition = LegacyKeep
			result.Reason = "live wrapper has exact artifact-owner tag"
		case !uncertain:
			result.Disposition = LegacyEligible
			result.Reason = "pre-lease orphan; complete sample has no wrapper with exact tag"
		}
		out = append(out, result)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PID < out[j].PID })
	return out
}

func legacyStart(platform, start string) ([2]uint64, bool) {
	var value [2]uint64
	if len(start) == 0 || len(start) > 32 {
		return value, false
	}
	parts := strings.Split(start, ".")
	if platform == "darwin" && len(parts) != 2 {
		return value, false
	}
	if platform == "linux" && len(parts) != 1 {
		return value, false
	}
	if platform != "darwin" && platform != "linux" {
		return value, false
	}
	for i, part := range parts {
		n, err := strconv.ParseUint(part, 10, 64)
		if err != nil || strconv.FormatUint(n, 10) != part {
			return value, false
		}
		value[i] = n
	}
	return value, value[0] > 0 && (platform != "darwin" || value[1] < 1_000_000)
}

// CleanupLegacyTitle acts only on an individually approved eligible row. It
// never samples new candidates, widens the approval set or removes metadata.
// Identity is rechecked immediately before each signal; unknown always stops.
func CleanupLegacyTitle(ctx context.Context, candidate LegacyTitleDecision, io LegacyCleanupIO, grace time.Duration) LegacyCleanupResult {
	result := LegacyCleanupResult{PID: candidate.PID, State: "kept"}
	if ctx == nil || io == nil || candidate.Disposition != LegacyEligible || candidate.PID <= 1 || candidate.PID > 1<<31-1 || candidate.Start == "" {
		return result
	}
	if grace <= 0 {
		grace = 250 * time.Millisecond
	}
	if grace > 2*time.Second {
		result.Err = errors.New("legacy cleanup grace exceeds two seconds")
		return result
	}
	same := func() bool {
		if err := ctx.Err(); err != nil {
			result.Err = err
			return false
		}
		identity, err := io.Identity(candidate.PID)
		if err != nil {
			result.Err = err
			return false
		}
		if identity != candidate.Start {
			result.State = "generation changed or unavailable"
			return false
		}
		return true
	}
	if !same() {
		return result
	}
	if err := io.Signal(candidate.PID, syscall.SIGTERM); err != nil {
		result.Err = fmt.Errorf("SIGTERM: %w", err)
		return result
	}
	result.State = "SIGTERM sent"
	if err := io.Sleep(ctx, grace); err != nil {
		result.Err = err
		return result
	}
	if !same() {
		return result
	}
	if err := io.Signal(candidate.PID, syscall.SIGKILL); err != nil {
		result.Err = fmt.Errorf("SIGKILL: %w", err)
		return result
	}
	result.State = "SIGKILL sent"
	return result
}

// CleanupApprovedLegacyTitles obtains exactly one fresh sample after approval,
// then intersects eligibility with approved PID/start/tag/agent generations.
// A wrapper born while the report awaited review revokes eligibility.
func CleanupApprovedLegacyTitles(ctx context.Context, approved []LegacyTitleDecision, policy LegacyPolicy, observe func(context.Context) (LegacySample, error), io LegacyCleanupIO, grace time.Duration) ([]LegacyCleanupResult, error) {
	if ctx == nil || observe == nil {
		return nil, errors.New("missing fresh legacy-title observation")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	fresh, err := observe(ctx)
	if err != nil {
		return nil, err
	}
	type key struct {
		pid               int
		start, tag, agent string
	}
	decisions := map[key]LegacyTitleDecision{}
	for _, candidate := range DecideLegacyTitles(fresh, policy) {
		decisions[key{candidate.PID, candidate.Start, candidate.Tag, candidate.Agent}] = candidate
	}
	seen := map[key]bool{}
	var results []LegacyCleanupResult
	for _, candidate := range approved {
		if candidate.Disposition != LegacyEligible {
			continue
		}
		k := key{candidate.PID, candidate.Start, candidate.Tag, candidate.Agent}
		if seen[k] {
			continue
		}
		seen[k] = true
		current, ok := decisions[k]
		if !ok || current.Disposition != LegacyEligible {
			results = append(results, LegacyCleanupResult{PID: candidate.PID, State: "approval no longer eligible in fresh sample"})
			continue
		}
		results = append(results, CleanupLegacyTitle(ctx, current, io, grace))
	}
	return results, nil
}
