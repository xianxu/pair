package sessioninventory

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
