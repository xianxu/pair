package mockserve

import (
	"fmt"
	"io"
)

// Check reports implementation progress without requiring or opening mock assets.
// Missing implementations are a successful report, not a production build gate.
func Check(dir string, out io.Writer) error {
	p, err := OpenProject(dir)
	if err != nil {
		return fmt.Errorf("check screens.json: %w", err)
	}
	defer p.Close()
	raw, err := p.readBounded("screens.json", MaxCatalogueBytes)
	if err != nil {
		return fmt.Errorf("check screens.json: %w", err)
	}
	c, err := ParseCatalogue(raw)
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, c.Report())
	return err
}
