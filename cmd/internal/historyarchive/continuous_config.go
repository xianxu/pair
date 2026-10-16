package historyarchive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ContinuousConfigName is the operator-written configuration for continuous
// archiving, under the archive store ($PAIR_DATA_DIR/archive/). Without it a
// continuous pass does nothing.
const ContinuousConfigName = "continuous.json"

// ContinuousConfig is where continuous archiving puts bundles and who can read
// them, plus the two roots the inventory cannot derive: the ops checkout and
// the workspace whose git repos supply sdlc runtime data.
// pair:m5-concept pure
type ContinuousConfig struct {
	Version    int      `json:"version"`
	Recipients []string `json:"recipients"`
	Sink       string   `json:"sink"`
	Catalog    string   `json:"catalog"`
	OpsRoot    string   `json:"ops_root,omitempty"`
	Workspace  string   `json:"workspace,omitempty"`
}

// LoadContinuousConfig reads path strictly. An absent file is (nil, nil):
// continuous archiving is not configured, which is not an error.
func LoadContinuousConfig(path string) (*ContinuousConfig, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var c ContinuousConfig
	if err := decodeStrict(raw, &c); err != nil {
		return nil, fmt.Errorf("continuous config: %w", err)
	}
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("continuous config: %w", err)
	}
	return &c, nil
}

func (c ContinuousConfig) validate() error {
	if c.Version != Version {
		return errors.New("unsupported version")
	}
	if err := validateRecipients(c.Recipients); err != nil {
		return err
	}
	for name, p := range map[string]string{"sink": c.Sink, "catalog": c.Catalog} {
		if !filepath.IsAbs(p) {
			return fmt.Errorf("%s must be an absolute path", name)
		}
	}
	for name, p := range map[string]string{"ops_root": c.OpsRoot, "workspace": c.Workspace} {
		if p != "" && !filepath.IsAbs(p) {
			return fmt.Errorf("%s must be an absolute path", name)
		}
	}
	return nil
}
