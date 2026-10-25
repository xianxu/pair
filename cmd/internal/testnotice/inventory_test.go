package testnotice

import (
	"strings"
	"testing"
)

func TestFindingsAuditedSitesKeepExactReceiverAndFile(t *testing.T) {
	audit := Audit{
		Projection: func(file, function string) bool {
			return file == "projection.go" && function == "Console.ObserveAttention"
		},
		Forward: func(file, function, method, arg string) bool {
			return file == "forward.go" && function == "Console.Warn" && method == "publishWarningLocked" && arg == "n"
		},
	}
	for _, tc := range []struct {
		name, file, source string
		want               int
	}{
		{"pointer receiver projection", "projection.go", `package p;func(c *Console)ObserveAttention(){c.attn.list=c.attn.list.Apply(fact,now)}`, 0},
		{"value receiver projection", "projection.go", `package p;func(c Console)ObserveAttention(){c.attn.list=c.attn.list.Apply(fact,now)}`, 0},
		{"wrong projection receiver", "projection.go", `package p;func(c Other)ObserveAttention(){c.attn.list=c.attn.list.Apply(fact,now)}`, 1},
		{"wrong projection file", "new.go", `package p;func(c *Console)ObserveAttention(){c.attn.list=c.attn.list.Apply(fact,now)}`, 1},
		{"audited forward", "forward.go", `package p;func(c *Console)Warn(n Notice){c.publishWarningLocked(n)}`, 0},
		{"wrong forward arg", "forward.go", `package p;func(c *Console)Warn(n Notice){c.publishWarningLocked(other)}`, 1},
		{"wrong forward receiver", "forward.go", `package p;func(c *Other)Warn(n Notice){c.publishWarningLocked(n)}`, 1},
		{"wrong forward file", "new.go", `package p;func(c *Console)Warn(n Notice){c.publishWarningLocked(n)}`, 1},
		{"nested closure inherits enclosing site", "forward.go", `package p;func(c *Console)Warn(n Notice){go func(){c.publishWarningLocked(n)}()}`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Findings(tc.file, []byte(tc.source), audit); len(got) != tc.want {
				t.Fatalf("findings=%v want count %d", got, tc.want)
			}
		})
	}
}

func TestFindingsDefaultsRejectAndPreservePositions(t *testing.T) {
	findings := Findings("new.go", []byte("package p\nfunc f(){console.Warn(Notice{Body: \"failure\"})}"), Audit{})
	if len(findings) != 1 || !strings.HasPrefix(findings[0], "new.go:2:") {
		t.Fatalf("findings=%v", findings)
	}
	if got := Findings("invalid.go", []byte("package"), Audit{}); len(got) != 1 {
		t.Fatalf("syntax error findings=%v", got)
	}
}

func TestInventoryRejections(t *testing.T) { CheckRejections(t, Audit{}) }
