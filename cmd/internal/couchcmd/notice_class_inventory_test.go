package couchcmd

import (
	"os"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testnotice"
)

// Scan the whole package so a warning in a new production file cannot escape
// classification. Read only local sources; policy and origin live elsewhere.
func TestNoticeClassInventory(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, finding := range testnotice.Findings(name, source, noticeClassAudit()) {
			t.Error(finding)
		}
	}
}

func noticeClassAuditedProjection(file, function string) bool {
	switch file + "/" + function {
	case "attention.go/waitWatcher.publishQuestions", "message_service.go/messageService.publishBlocked", "notify_service.go/messageService.handleNotify":
		return true
	}
	return false
}

// These are the existing audited computed paths, not a producer policy table.
// Every other producer must show its Class at the warning call.
func noticeClassAuditedForward(file, function, method, arg string) bool {
	if file == "run.go" && function == "beginConsole" && method == "Warn" && arg == "notice" {
		return true
	}
	if file == "hub_move.go" && function == "messageService.installHubMoves" && method == "Warn" && arg == "warning" {
		return true
	}
	return false
}

func TestNoticeClassInventoryRejectsNewUnclassifiedProducers(t *testing.T) {
	testnotice.CheckRejections(t, noticeClassAudit())
}

func noticeClassAudit() testnotice.Audit {
	return testnotice.Audit{Projection: noticeClassAuditedProjection, Forward: noticeClassAuditedForward}
}
