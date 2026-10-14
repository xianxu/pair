package wrapcmd

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

// Evidence contains no terminal handles or hyperlink metadata and has a hard
// 16KiB screen budget. Capturing it performs no disk or network IO.
type automaticInputEvidence struct {
	Report                             couchmessage.AutomaticInputStall
	Width, Height, CursorX, CursorY    int
	Complete, CursorVisible, Truncated bool
	Screen                             []string
}

func captureAutomaticInputEvidence(report couchmessage.AutomaticInputStall, s terminalSnapshot) automaticInputEvidence {
	e := automaticInputEvidence{Report: report, Width: s.Width, Height: s.Height, CursorX: s.Cursor.X, CursorY: s.Cursor.Y, Complete: s.Complete, CursorVisible: s.CursorVisible}
	left := 16384
	start := max(0, s.Height-40)
	e.Truncated = start > 0
	for y := start; y < s.Height; y++ {
		if s.Width <= 0 || s.Width > len(s.Cells) || y > (len(s.Cells)-s.Width)/s.Width {
			e.Truncated = true
			break
		}
		marker := -1
		if s.CursorVisible && y == s.Cursor.Y {
			marker = s.Cursor.X
		}
		line := notifyosc.Sanitize([]byte(renderTailRow(s.Cells[y*s.Width:(y+1)*s.Width], marker)))
		if len(line) > left {
			line = strings.ToValidUTF8(line[:left], "")
			e.Truncated = true
		}
		e.Screen = append(e.Screen, line)
		left -= len(line)
		if left == 0 {
			e.Truncated = e.Truncated || y < s.Height-1
			break
		}
	}
	return e
}
func runAutomaticInputReporter(ctx context.Context, p *proxy, b couchmessage.Binding, namespace string, queue <-chan automaticInputEvidence) {
	socket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		p.traceWrap("automatic-input-report-failed", map[string]any{"error": err.Error()})
		return
	}
	for {
		select {
		case <-ctx.Done():
			// Drain accepted diagnostic evidence, but never retry network effects at shutdown.
			for {
				select {
				case evidence := <-queue:
					p.traceWrap("automatic-input-stall", map[string]any{"evidence": evidence, "report_cancelled": true})
				default:
					if drops := p.automaticReportDrops.Swap(0); drops > 0 {
						p.traceWrap("automatic-input-report-failed", map[string]any{"dropped": drops, "error": "report queue full"})
					}
					return
				}
			}

		case evidence := <-queue:
			if drops := p.automaticReportDrops.Swap(0); drops > 0 {
				p.traceWrap("automatic-input-report-failed", map[string]any{"dropped": drops, "error": "report queue full"})
			}
			p.traceWrap("automatic-input-stall", map[string]any{"evidence": evidence, "report_cancelled": ctx.Err() != nil})
			if ctx.Err() != nil {
				continue
			}
			raw, err := json.Marshal(evidence.Report)
			if err == nil {
				request := couchmessage.Request{Op: "automatic-input-stall", Scope: b.Scope, Tag: b.Tag, Session: b.Session, Nonce: b.Nonce, ID: couchmessage.AutomaticInputStallID(b, evidence.Report.Sequence), Body: string(raw)}
				callCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				var response couchmessage.Response
				err = couchmessage.Call(callCtx, socket, request, &response)
				cancel()
				if err == nil && response.Code != "ok" {
					p.traceWrap("automatic-input-report-failed", map[string]any{"sequence": evidence.Report.Sequence, "code": response.Code, "error": response.Error})
					continue
				}
			}
			if err != nil {
				p.traceWrap("automatic-input-report-failed", map[string]any{"sequence": evidence.Report.Sequence, "error": err.Error()})
			}
		}
	}
}

func (p *proxy) startAutomaticInputReporter(ctx context.Context, b couchmessage.Binding, namespace string, workers *sync.WaitGroup) {
	queue := make(chan automaticInputEvidence, 1)
	p.automaticReport = func(report couchmessage.AutomaticInputStall, snapshot terminalSnapshot) {
		evidence := captureAutomaticInputEvidence(report, snapshot)
		select {
		case <-ctx.Done():
			return
		default:
		}
		select {
		case queue <- evidence:
		case <-ctx.Done():
		default:
			p.automaticReportDrops.Add(1)
		}
	}
	workers.Add(1)
	go func() { defer workers.Done(); runAutomaticInputReporter(ctx, p, b, namespace, queue) }()
}
