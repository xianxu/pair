package vmcmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

type Lane string

const (
	TrainLane Lane = "train"
	AdhocLane Lane = "adhoc"
)

type RunRequest struct {
	Checkout  string
	Lane      Lane
	Timeout   time.Duration
	Artifacts string
	Args      []string
}

// ParseRun validates a request without consulting the host or interpreting argv.
func ParseRun(args []string) (RunRequest, error) {
	r := RunRequest{Lane: AdhocLane, Timeout: 20 * time.Minute}
	seen := map[string]bool{}
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			if !seen["--checkout"] {
				return RunRequest{}, fmt.Errorf("--checkout is required")
			}
			if i+1 == len(args) || args[i+1] == "" {
				return RunRequest{}, fmt.Errorf("command argv is required after --")
			}
			for _, arg := range args[i+1:] {
				if strings.ContainsRune(arg, '\x00') {
					return RunRequest{}, fmt.Errorf("command argv contains NUL")
				}
			}
			r.Args = append([]string(nil), args[i+1:]...)
			return r, nil
		}
		name, value, inline := strings.Cut(args[i], "=")
		switch name {
		case "--checkout", "--lane", "--timeout", "--artifacts":
		default:
			return RunRequest{}, fmt.Errorf("unknown run option %q", name)
		}
		if seen[name] {
			return RunRequest{}, fmt.Errorf("duplicate run option %s", name)
		}
		seen[name] = true
		if !inline {
			i++
			if i == len(args) || args[i] == "--" || strings.HasPrefix(args[i], "--") {
				return RunRequest{}, fmt.Errorf("%s requires a value", name)
			}
			value = args[i]
		}
		if value == "" {
			return RunRequest{}, fmt.Errorf("%s requires a nonempty value", name)
		}
		switch name {
		case "--checkout", "--artifacts":
			if err := validateRunPath(value); err != nil {
				return RunRequest{}, fmt.Errorf("%s: %w", name, err)
			}
			if name == "--checkout" {
				r.Checkout = value
			} else {
				r.Artifacts = value
			}
		case "--lane":
			r.Lane = Lane(value)
			if r.Lane != TrainLane && r.Lane != AdhocLane {
				return RunRequest{}, fmt.Errorf("lane must be train or adhoc")
			}
		case "--timeout":
			d, err := time.ParseDuration(value)
			if err != nil || d <= 0 {
				return RunRequest{}, fmt.Errorf("timeout must be a positive duration")
			}
			r.Timeout = d
		}
	}
	return RunRequest{}, fmt.Errorf("run requires -- followed by command argv")
}

func validateRunPath(value string) error {
	if !filepath.IsAbs(value) || filepath.Clean(value) != value {
		return fmt.Errorf("path must be absolute and lexically clean")
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("path contains a control character")
		}
	}
	return nil
}
