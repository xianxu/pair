package wrapcmd

import (
	"errors"
	"os"

	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
)

func (p *proxy) diagnosticLogOptions() diagnosticlog.Options {
	if p.diagnosticOptions != nil {
		return p.diagnosticOptions()
	}
	return diagnosticlog.EnvironmentOptions(os.Getenv)
}

func (p *proxy) closeDebug() error {
	p.debugMu.Lock()
	p.debugClosed = true
	w := p.debugWriter
	p.debugMu.Unlock()
	return w.Close()
}

func (p *proxy) openWrapEvents(argv []string) {
	if p.wrapEventsPath == "" || os.Getenv("PAIR_WRAP_EVENTS") == "0" {
		return
	}
	f, err := diagnosticlog.OpenBuffered(p.wrapEventsPath, p.diagnosticLogOptions())
	if err != nil {
		p.debug("WRAP-EVENTS-open-fail", err.Error())
		return
	}
	p.wrapEventsFD = f
	p.traceWrap("start", map[string]any{"pid": os.Getpid(), "argv": append([]string(nil), argv...)})
}

func (p *proxy) closeWrapEvents() error {
	if p.wrapEventsFD == nil {
		return nil
	}
	p.traceWrap("proxy-defer-close", nil)
	return p.wrapEventsFD.Close()
}

func (p *proxy) scrollbackOffset() int64 {
	if p.capture != nil {
		return p.capture.Offset()
	}
	return p.scrollbackBytes
}

func (p *proxy) closeCapture() error {
	err := p.capture.Close()
	if p.scrollbackFD != nil {
		err = errors.Join(err, p.scrollbackFD.Close())
	}
	if p.eventsFD != nil {
		err = errors.Join(err, p.eventsFD.Close())
	}
	return err
}
