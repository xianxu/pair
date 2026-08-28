package sessioninventory

import (
	"bytes"
	"encoding/json"
)

type ForestProjection struct {
	Forests     []Forest     `json:"forests"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// RenderForestProjection is M1's stable internal forest oracle. The complete
// public schema-v1 renderer is added only when correlation fields exist.
func RenderForestProjection(inventory Inventory) ([]byte, error) {
	canonical := SortInventory(inventory)
	projection := ForestProjection{Forests: canonical.Forests, Diagnostics: canonical.Diagnostics}
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(projection); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
