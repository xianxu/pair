// Package vt provides a virtual terminal implementation.
// SKIP: Fix typecheck errors - function signature mismatches and undefined types
package vt

import (
	"bytes"
	"encoding/base64"
	"image/color"
	"io"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// handleOsc handles an OSC escape sequence.
func (e *Emulator) handleOsc(cmd int, data []byte) {
	e.flushGrapheme() // Flush any pending grapheme before handling OSC sequences.
	if len(data) > e.limits.StringBytes {
		return
	}
	if e.handlePairEffect(cmd, data) {
		return
	}
	if !e.handlers.handleOsc(cmd, data) {
		e.logf("unhandled sequence: OSC %q", data)
	}
}

func (e *Emulator) handleTitle(cmd int, data []byte) {
	parts := bytes.SplitN(data, []byte{';'}, 2)
	if len(parts) != 2 {
		// Invalid, ignore
		return
	}
	switch cmd {
	case 0: // Set window title and icon name
		if len(parts[1]) > e.limits.MetadataBytes {
			return
		}
		name := string(parts[1])
		e.iconName, e.title = name, name
		if e.cb.Title != nil {
			e.cb.Title(name)
		}
		if e.cb.IconName != nil {
			e.cb.IconName(name)
		}
	case 1: // Set icon name
		if len(parts[1]) > e.limits.MetadataBytes {
			return
		}
		name := string(parts[1])
		e.iconName = name
		if e.cb.IconName != nil {
			e.cb.IconName(name)
		}
	case 2: // Set window title
		if len(parts[1]) > e.limits.MetadataBytes {
			return
		}
		name := string(parts[1])
		e.title = name
		if e.cb.Title != nil {
			e.cb.Title(name)
		}
	}
}

func (e *Emulator) handleDefaultColor(cmd int, data []byte) {
	if cmd != 10 && cmd != 11 && cmd != 12 &&
		cmd != 110 && cmd != 111 && cmd != 112 {
		// Invalid, ignore
		return
	}

	parts := bytes.SplitN(data, []byte{';'}, 2)
	if len(parts) == 0 {
		// Invalid, ignore
		return
	}

	cb := func(c color.Color) {
		switch cmd {
		case 10, 110: // Foreground color
			e.SetForegroundColor(c)
		case 11, 111: // Background color
			e.SetBackgroundColor(c)
		case 12, 112: // Cursor color
			e.SetCursorColor(c)
		}
	}

	switch len(parts) {
	case 1: // Reset color
		cb(nil)
	case 2: // Set/Query color
		arg := string(parts[1])
		if arg == "?" {
			var xrgb ansi.XRGBColor
			switch cmd {
			case 10: // Query foreground color
				xrgb.Color = e.ForegroundColor()
				if xrgb.Color != nil {
					io.WriteString(e.replies(), ansi.SetForegroundColor(xrgb.String())) //nolint:errcheck,gosec
				}
			case 11: // Query background color
				xrgb.Color = e.BackgroundColor()
				if xrgb.Color != nil {
					io.WriteString(e.replies(), ansi.SetBackgroundColor(xrgb.String())) //nolint:errcheck,gosec
				}
			case 12: // Query cursor color
				xrgb.Color = e.CursorColor()
				if xrgb.Color != nil {
					io.WriteString(e.replies(), ansi.SetCursorColor(xrgb.String())) //nolint:errcheck,gosec
				}
			}
		} else if c := ansi.XParseColor(arg); c != nil {
			cb(c)
		}
	}
}

func (e *Emulator) handleWorkingDirectory(cmd int, data []byte) {
	if cmd != 7 {
		// Invalid, ignore
		return
	}

	// The data is the working directory path.
	parts := bytes.SplitN(data, []byte{';'}, 2)
	if len(parts) != 2 {
		// Invalid, ignore
		return
	}

	if len(parts[1]) > e.limits.MetadataBytes {
		return
	}
	path := string(parts[1])
	e.cwd = path

	if e.cb.WorkingDirectory != nil {
		e.cb.WorkingDirectory(path)
	}
}

func (e *Emulator) handleHyperlink(cmd int, data []byte) {
	parts := bytes.SplitN(data, []byte{';'}, 3)
	if len(parts) != 3 || cmd != 8 {
		// Invalid, ignore
		return
	}

	if len(parts[1])+len(parts[2]) > e.limits.LinkBytes {
		return
	}
	e.scr.cur.Link.Params = string(parts[1])
	e.scr.cur.Link.URL = string(parts[2])
}

func (e *Emulator) handlePairEffect(cmd int, data []byte) bool {
	switch cmd {
	case 52:
		p := bytes.SplitN(data, []byte{';'}, 3)
		if len(p) != 3 {
			return true
		}
		if len(p[1]) > 16 {
			return true
		}
		selection := string(p[1])
		for _, c := range selection {
			if !strings.ContainsRune("cpsq01234567", c) {
				return true
			}
		}
		if string(p[2]) == "?" {
			if e.cb.ClipboardQuery != nil {
				e.cb.ClipboardQuery(selection)
			}
			return true
		}
		decoded, err := base64.StdEncoding.DecodeString(string(p[2]))
		if err == nil && e.cb.ClipboardWrite != nil {
			e.cb.ClipboardWrite(selection, decoded)
		}
		return true
	case 9:
		p := bytes.SplitN(data, []byte{';'}, 2)
		if len(p) == 2 && len(p[1]) <= e.limits.MetadataBytes && e.cb.Notification != nil {
			e.cb.Notification("", string(p[1]))
		}
		return true
	case 777:
		p := bytes.SplitN(data, []byte{';'}, 4)
		if len(p) == 4 && string(p[1]) == "notify" && len(p[2]) <= e.limits.MetadataBytes && len(p[3]) <= e.limits.MetadataBytes && e.cb.Notification != nil {
			e.cb.Notification(string(p[2]), string(p[3]))
		}
		return true
	}
	return false
}
