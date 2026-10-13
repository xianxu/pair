package sessioninventory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
)

type ConformanceStatus string

const (
	ConformanceOK   ConformanceStatus = "ok"
	ConformanceSkip ConformanceStatus = "skip"
	ConformanceFail ConformanceStatus = "fail"
)

type ConformanceAgent struct {
	Agent       Agent             `json:"agent"`
	Status      ConformanceStatus `json:"status"`
	Nodes       int               `json:"nodes"`
	Roots       int               `json:"roots"`
	Diagnostics []DiagnosticCode  `json:"diagnostics"`
}

type ConformanceReport struct {
	Agents []ConformanceAgent `json:"agents"`
}

func RunConformance(runtime Runtime, agents ...Agent) (ConformanceReport, error) {
	agents = append([]Agent(nil), agents...)
	sort.Slice(agents, func(i, j int) bool { return agents[i] < agents[j] })
	report := ConformanceReport{Agents: make([]ConformanceAgent, 0, len(agents))}
	var failures []error
	for _, agent := range agents {
		entry := ConformanceAgent{Agent: agent, Status: ConformanceOK, Diagnostics: []DiagnosticCode{}}
		available, unreadable := conformanceAvailability(runtime, agent)
		if unreadable != nil {
			entry.Status = ConformanceFail
			entry.Diagnostics = []DiagnosticCode{DiagnosticStorageUnreadable}
			failures = append(failures, fmt.Errorf("%s: installed storage unreadable", agent))
			report.Agents = append(report.Agents, entry)
			continue
		}
		if !available {
			entry.Status = ConformanceSkip
			entry.Diagnostics = []DiagnosticCode{DiagnosticConformanceNoSample}
			report.Agents = append(report.Agents, entry)
			continue
		}
		result := ScannerForAgent(agent)(runtime)
		inventory := BuildForest(result.Facts)
		inventory.Diagnostics = append(inventory.Diagnostics, result.Diagnostics...)
		entry.Roots, entry.Nodes = forestCounts(inventory.Forests)
		entry.Diagnostics = diagnosticCodes(inventory.Diagnostics)
		if entry.Nodes == 0 || hasConformanceFailure(entry.Diagnostics) {
			entry.Status = ConformanceFail
			failures = append(failures, fmt.Errorf("%s: native schema drift (%v)", agent, entry.Diagnostics))
		}
		report.Agents = append(report.Agents, entry)
	}
	return report, errors.Join(failures...)
}

func RenderConformance(report ConformanceReport) ([]byte, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(report); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// ValidateCodexLifecycleConformance checks the exact production envelopes that
// notification authority consumes. At least one keyed successful turn must be
// present, and every accepted terminal must follow its same-root opener.
func ValidateCodexLifecycleConformance(events []NativeEventFact) error {
	started := map[string]uint64{}
	completed := 0
	sort.Slice(events, func(i, j int) bool { return events[i].Position < events[j].Position })
	for _, fact := range events {
		key := fact.RootNodeID + "\x00" + fact.Event.TurnID
		switch fact.Event.SourceKind {
		case "event_msg.task_started":
			if fact.Event.TurnID == "" || fact.Event.Timestamp.IsZero() {
				return errors.New("Codex task_started lacks turn identity or timestamp")
			}
			started[key] = fact.Position
		case "event_msg.task_complete", "event_msg.turn_aborted":
			position, ok := started[key]
			if fact.Event.TurnID == "" || fact.Event.Timestamp.IsZero() || !ok || fact.Position <= position {
				return errors.New("Codex terminal lacks a prior same-turn opener")
			}
			if fact.Event.SourceKind == "event_msg.task_complete" {
				completed++
			}
		}
	}
	if completed == 0 {
		return errors.New("Codex lifecycle conformance found no completed keyed turn")
	}
	return nil
}

func conformanceAvailability(runtime Runtime, agent Agent) (bool, error) {
	available := false
	for _, root := range runtime.NativeRoots(agent) {
		files, err := runtime.ListFiles(root)
		if errors.Is(err, ErrStorageAbsent) {
			continue
		}
		var issues *ListingIssuesError
		if errors.As(err, &issues) {
			if len(files) != 0 {
				available = true
			}
			continue
		}
		if err != nil {
			return false, err
		}
		if len(files) != 0 {
			available = true
		}
	}
	return available, nil
}

func ScannerForAgent(agent Agent) ScannerFunc {
	switch agent {
	case AgentClaude:
		return ScanClaude
	case AgentCodex:
		return ScanCodex
	case AgentAgy:
		return ScanAgy
	case AgentMuse:
		return ScanMuse
	case AgentQoder:
		return ScanQoder
	case AgentGrok:
		return ScanGrok
	default:
		return func(Runtime) ScanResult {
			return ScanResult{Diagnostics: []Diagnostic{diagnostic(DiagnosticSchemaNearMiss, agent, nil, "unsupported agent")}}
		}
	}
}

func forestCounts(forests []Forest) (roots, nodes int) {
	var countNode func(Node) int
	countNode = func(node Node) int {
		count := 1
		for _, child := range node.Children {
			count += countNode(child)
		}
		return count
	}
	for _, forest := range forests {
		roots += len(forest.Roots)
		for _, root := range forest.Roots {
			nodes += countNode(root)
		}
		for _, orphan := range forest.Orphans {
			nodes += countNode(orphan)
		}
	}
	return roots, nodes
}

func diagnosticCodes(diagnostics []Diagnostic) []DiagnosticCode {
	unique := make(map[DiagnosticCode]struct{})
	for _, diagnostic := range diagnostics {
		unique[diagnostic.Code] = struct{}{}
	}
	codes := make([]DiagnosticCode, 0, len(unique))
	for code := range unique {
		codes = append(codes, code)
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
	return codes
}

func hasConformanceFailure(codes []DiagnosticCode) bool {
	for _, code := range codes {
		switch code {
		case DiagnosticStorageUnreadable, DiagnosticSchemaNearMiss:
			return true
		}
	}
	return false
}

// RecoveryConformanceAgent reports contract coverage without exposing native
// paths, identifiers, or authored text. Skip never means the contract passed.
type RecoveryConformanceAgent struct {
	Agent         Agent             `json:"agent"`
	Status        ConformanceStatus `json:"status"`
	Reason        string            `json:"reason"`
	Samples       int               `json:"samples"`
	Interactive   int               `json:"interactive"`
	AuthoredTails int               `json:"authored_tails"`
	ParserVersion string            `json:"parser_version"`
}
type RecoveryConformanceReport struct {
	Agents []RecoveryConformanceAgent `json:"agents"`
}

// RunRecoveryConformance exercises the production recovery interpreter through
// its bounded snapshot reader. No capture epoch is invented and no selection or
// native binding is authorized by this diagnostic. Run on contract changes and
// with the existing monthly native-shape conformance check.
func RunRecoveryConformance(ctx context.Context, runtime Runtime, agents ...Agent) (RecoveryConformanceReport, error) {
	report := RecoveryConformanceReport{}
	if ctx == nil {
		return report, errors.New("recovery conformance requires context")
	}
	ctx, cancel := context.WithTimeout(ctx, RecoveryScanTimeout)
	defer cancel()
	agents = append([]Agent(nil), agents...)
	sort.Slice(agents, func(i, j int) bool { return agents[i] < agents[j] })
	var failures []error
	var budget RecoveryEvidence
	metadata := 0
	io, bounded := runtime.(RecoveryRuntime)
	for _, agent := range agents {
		result := RecoveryConformanceAgent{Agent: agent, Status: ConformanceSkip, Reason: "absent-sample", ParserVersion: RecoveryParserVersion}
		if agent != AgentCodex && agent != AgentClaude {
			result.Reason = "unsupported-interactive-provenance"
			report.Agents = append(report.Agents, result)
			continue
		}
		fail := func(reason string) { result.Status = ConformanceFail; result.Reason = reason }
		if !bounded {
			fail("bounded-recovery-io-unavailable")
		} else {
			for _, root := range runtime.NativeRoots(agent) {
				if metadata >= RecoveryMetadataLimit {
					fail("metadata-limit")
					break
				}
				samplesBefore := result.Samples
				count, err := io.VisitRecoveryFiles(ctx, root, RecoveryMetadataLimit-metadata, func(entry FileEntry) error {
					if entry.Artifact.StorageRoot != root.Name {
						return ErrPathEscape
					}
					if _, ok := observeEntry(agent, entry); !ok {
						return nil
					}
					result.Samples++
					parser := newRecoveryParser(agent, entry)
					err := readRecoverySnapshot(ctx, io, entry, &budget, func(records []FramedJSONLRecord) error {
						for _, record := range records {
							if err := parser.add(record); err != nil {
								return err
							}
							events, _ := NormalizeNativeEvent(agent, record.Bytes)
							if err := validateRecoveryConformanceTail(parser.c, events); err != nil {
								return err
							}
						}
						return nil
					})
					if err != nil {
						return err
					}
					candidate, _, err := parser.finish(nil)
					if err != nil {
						return err
					}
					if candidate.Interactive.State == InteractiveNo {
						return nil
					}
					// Unknown provenance on an otherwise recognized root is drift, not an
					// absent interactive sample. In particular, disappearing CLI fields fail.
					if candidate.Interactive.State != InteractiveYes || !candidate.RootEligible {
						return errors.New("recovery provenance drift")
					}
					result.Interactive++
					if candidate.Incomplete || candidate.CWD == "" {
						return errors.New("recovery cwd or authored envelope drift")
					}
					if len(candidate.Messages) == 0 {
						return nil
					}
					if candidate.SemanticLast.IsZero() {
						return errors.New("recovery activity drift")
					}
					for _, message := range candidate.Messages {
						if message.Role == EventAssistant && message.Complete && message.Text != "" {
							if message.Timestamp.IsZero() || message.CWD != candidate.CWD {
								return errors.New("recovery authored tail drift")
							}
							result.AuthoredTails++
							break
						}
					}
					return nil
				})
				if count < 0 || count > RecoveryMetadataLimit-metadata {
					fail("invalid-metadata-count")
					break
				}
				metadata += count
				if (errors.Is(err, ErrStorageAbsent) || errors.Is(err, fs.ErrNotExist)) && count == 0 && result.Samples == samplesBefore {
					continue
				}
				if err != nil {
					fail("recovery-contract-or-snapshot-failed")
					break
				}
			}
		}
		if result.Status == ConformanceFail {
			failures = append(failures, fmt.Errorf("%s: %s", agent, result.Reason))
		} else if result.AuthoredTails > 0 {
			result.Status = ConformanceOK
			result.Reason = "provenance-cwd-activity-authored-tail"
		} else if result.Samples > 0 {
			result.Reason = "no-authored-interactive-sample"
		}
		report.Agents = append(report.Agents, result)
	}
	return report, errors.Join(failures...)
}

// The established event normalizer is an independent witness that a sample
// contains authored text; recovery must not silently turn it into an empty tail.
func validateRecoveryConformanceTail(candidate RecoveryCandidate, events []NativeEvent) error {
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if (event.Kind == EventAssistant || event.Kind == EventOperator) && event.Text != "" {
			if len(candidate.Messages) == 0 {
				return errors.New("recovery lost recognized authored tail")
			}
			last := candidate.Messages[len(candidate.Messages)-1]
			if last.Text != event.Text || last.Role != event.Kind || !last.Complete {
				return errors.New("recovery authored tail disagrees with native event contract")
			}
			break
		}
	}
	return nil
}
