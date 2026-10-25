package couchcmd

import (
	"os"
	"strings"
	"testing"
)

func TestServiceProducerNoticeClasses(t *testing.T) {
	for file, class := range map[string]string{
		"hub_move.go":               "hubConflictNotices",
		"renicer.go":                "NoticeAgentPriorityWarning",
		"scratch.go":                "NoticeScratchMetadataWarning",
		"lifecyclefailure_queue.go": "NoticeRemoteLifecycleDiagnostic",
		"hubs.go":                   "NoticeHubStartFailed",
	} {
		t.Run(file, func(t *testing.T) {
			body, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(body), class) {
				t.Fatalf("producer lacks declared class %s", class)
			}
		})
	}
}
