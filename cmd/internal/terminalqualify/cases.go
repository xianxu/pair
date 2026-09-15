package terminalqualify

import "github.com/charmbracelet/x/vt"

// Case expectations are literal protocol observations, never derived by running
// the candidate. Uncovered identifies an obligation this tool cannot establish.
type Case struct {
	ID, Capability, Source, Input string
	Expected                      Observation
	Action                        func(*vt.Emulator)
	Split                         bool
	Uncovered                     string
	Width, Height                 int
}

// SplitInputs includes whole, every single byte boundary, and byte-at-a-time.
func SplitInputs(input string) [][]string {
	out := [][]string{{input}}
	for i := 1; i < len(input); i++ {
		out = append(out, []string{input[:i], input[i:]})
	}
	if len(input) > 1 {
		bytes := make([]string, len(input))
		for i := range len(input) {
			bytes[i] = input[i : i+1]
		}
		out = append(out, bytes)
	}
	return out
}
