package couchtty

import "fmt"

// TabStyle selects the launch-local chrome packing policy.
type TabStyle uint8

const (
	TabStyleMultiline TabStyle = iota
	TabStyleCollapse
	DefaultTabStyle = TabStyleCollapse
)

// ParseTabStyle accepts the public flag values; empty uses the launch default.
func ParseTabStyle(value string) (TabStyle, error) {
	switch value {
	case "":
		return DefaultTabStyle, nil
	case "collapse":
		return TabStyleCollapse, nil
	case "multiline":
		return TabStyleMultiline, nil
	default:
		return 0, fmt.Errorf("invalid tab style %q: use collapse or multiline", value)
	}
}
