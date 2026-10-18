package zellijreply

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Client is one native list-clients row. PaneID retains its native role prefix.
type Client struct {
	ID                     int
	PaneID, RunningCommand string
}

func ParseClients(data []byte) ([]Client, error) {
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || strings.Join(strings.Fields(lines[0]), " ") != "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND" {
		return nil, fmt.Errorf("invalid Zellij clients header")
	}
	out := make([]Client, 0, len(lines)-1)
	seen := map[int]bool{}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Only split the first two columns; command arguments are an opaque remainder.
		fields := strings.Fields(line)
		if len(fields) < 3 {
			return nil, fmt.Errorf("invalid Zellij client row")
		}
		id, err := decimalID(fields[0])
		if err != nil || seen[id] {
			return nil, fmt.Errorf("invalid or duplicate client identity")
		}
		_, _, err = qualifiedPaneID(fields[1])
		if err != nil {
			return nil, err
		}
		rest := strings.TrimLeftFunc(line, unicode.IsSpace)
		rest = strings.TrimLeftFunc(rest[len(fields[0]):], unicode.IsSpace)
		rest = strings.TrimLeftFunc(rest[len(fields[1]):], unicode.IsSpace)
		seen[id] = true
		out = append(out, Client{ID: id, PaneID: fields[1], RunningCommand: strings.TrimRightFunc(rest, unicode.IsSpace)})
	}
	return out, nil
}

func decimalID(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("missing identity")
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid identity")
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid identity")
	}
	return n, nil
}
func qualifiedPaneID(s string) (int, bool, error) {
	if strings.HasPrefix(s, "terminal_") {
		n, e := decimalID(strings.TrimPrefix(s, "terminal_"))
		return n, false, e
	}
	if strings.HasPrefix(s, "plugin_") {
		n, e := decimalID(strings.TrimPrefix(s, "plugin_"))
		return n, true, e
	}
	return 0, false, fmt.Errorf("invalid qualified pane identity")
}
