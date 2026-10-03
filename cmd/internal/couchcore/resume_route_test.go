package couchcore

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
)

// declaredResumeCodes is every constant of type ResumeDiagnosticCode written in
// resume.go, found by parsing the file rather than by a hand list -- a code
// added later reaches this test without anyone remembering to add it here.
func declaredResumeCodes(t *testing.T) map[ResumeDiagnosticCode]bool {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "resume.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	codes := map[ResumeDiagnosticCode]bool{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			value := spec.(*ast.ValueSpec)
			ident, ok := value.Type.(*ast.Ident)
			if !ok || ident.Name != "ResumeDiagnosticCode" {
				continue
			}
			for _, lit := range value.Values {
				basic, ok := lit.(*ast.BasicLit)
				if !ok || basic.Kind != token.STRING {
					t.Fatalf("ResumeDiagnosticCode constant with a non-literal value: %#v", lit)
				}
				codes[ResumeDiagnosticCode(strings.Trim(basic.Value, `"`))] = true
			}
		}
	}
	if len(codes) == 0 {
		t.Fatal("parsed no ResumeDiagnosticCode constants; the enumeration proves nothing")
	}
	return codes
}

func TestResumeRebootAdviceClassifiesEveryDeclaredCode(t *testing.T) {
	declared := declaredResumeCodes(t)
	for code := range declared {
		if _, ok := ResumeRebootAdvice[code]; !ok {
			t.Errorf("%s is declared but ResumeRebootAdvice does not say whether it names reboot", code)
		}
	}
	for code := range ResumeRebootAdvice {
		if !declared[code] {
			t.Errorf("ResumeRebootAdvice classifies %s, which resume.go does not declare", code)
		}
	}
}

// The Spec's split, as an independent literal: reboot is named only when the
// transcript cannot come back; a transient refusal says to retry instead.
func TestResumeRebootAdviceValues(t *testing.T) {
	want := map[ResumeDiagnosticCode]bool{
		ResumePathMissing:        true,
		ResumeProfileMissing:     true,
		ResumeProfileInvalid:     true,
		ResumeAgentUnsupported:   true,
		ResumeBindingUnbound:     true,
		ResumeBindingRootMissing: true,
		ResumeBindingAmbiguous:   true,
		ResumeTombstoned:         true,
		ResumeSessionGone:        true,
		ResumeNoSurvivor:         true,
		ResumeLive:               false,
		ResumeUnknown:            false,
		ResumeParking:            false,
		ResumeStarting:           false,
		ResumeBindingProvisional: false,
		ResumeNotDetached:        false,
		ResumeNotRunning:         false,
		ResumeSurvivorsAmbiguous: false,
	}
	if len(want) != len(ResumeRebootAdvice) {
		t.Errorf("advice has %d codes, the Spec lists %d", len(ResumeRebootAdvice), len(want))
	}
	for code, names := range want {
		got, ok := ResumeRebootAdvice[code]
		if !ok || got != names {
			t.Errorf("%s: advice %v (present %v), want %v", code, got, ok, names)
		}
	}
}

func TestWithRebootAdviceWrapsAndKeepsTheCode(t *testing.T) {
	cannot := withRebootAdvice(&ResumeRefusal{Code: ResumeBindingUnbound, Diagnostic: "no conversation"})
	if !strings.Contains(cannot.Error(), "reboot") || ResumeDiagnosticOf(cannot) != ResumeBindingUnbound {
		t.Fatalf("unrecoverable refusal: %v (code %q)", cannot, ResumeDiagnosticOf(cannot))
	}
	transient := withRebootAdvice(fmt.Errorf("wrapped: %w", &ResumeRefusal{Code: ResumeStarting, Diagnostic: "busy"}))
	if strings.Contains(transient.Error(), "reboot") {
		t.Fatalf("transient refusal names reboot: %v", transient)
	}
	plain := errors.New("plain")
	if withRebootAdvice(plain) != plain || withRebootAdvice(nil) != nil {
		t.Fatal("an error without a resume code must pass through untouched")
	}
}

// expectedResumeRoute is the plan's route table written as its own literal.
func expectedResumeRoute(in ResumeRouteInput) ResumeRoute {
	if in.Slot && !in.HasRecord {
		return ResumeRouteSlot
	}
	if in.HasRecord {
		switch in.Continuation {
		case checkpoint.Failed, checkpoint.Running:
			return ResumeRouteContinuation
		case checkpoint.Pending:
			return ResumeRouteRecover
		}
	}
	if in.Slot {
		return ResumeRouteSlot
	}
	if in.HasRecord && in.RecoveryRecover {
		return ResumeRouteRecover
	}
	return ResumeRouteThread
}

func TestChooseResumeRoute(t *testing.T) {
	phases := append(checkpoint.AllPhases(), "")
	for _, slot := range []bool{false, true} {
		for _, hasRecord := range []bool{false, true} {
			for _, phase := range phases {
				for _, recover := range []bool{false, true} {
					in := ResumeRouteInput{Slot: slot, HasRecord: hasRecord, Continuation: phase, RecoveryRecover: recover}
					if !hasRecord && (phase != "" || recover) {
						continue // no record carries no continuation and no recovery verdict
					}
					want := expectedResumeRoute(in)
					if want == 0 {
						t.Fatalf("%+v: the expected table has no route", in)
					}
					if got := ChooseResumeRoute(in); got != want {
						t.Errorf("%+v: route %v, want %v", in, got, want)
					}
				}
			}
		}
	}
}
