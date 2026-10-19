//go:build !darwin && !linux

package execalarm

import "fmt"

// Run is unavailable on platforms without the supported kernel timer API.
func Run(args []string) error {
	if _, _, err := parse(args); err != nil {
		return err
	}
	return fmt.Errorf("exec-alarm: kernel alarm unsupported on this platform")
}
