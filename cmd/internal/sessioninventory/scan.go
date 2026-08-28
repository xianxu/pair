package sessioninventory

// InventoryWithRuntime runs facts-only scanners through the shared IO seam,
// then performs all coalescing and tree construction in the pure core.
func InventoryWithRuntime(runtime Runtime, scanners ...Scanner) Inventory {
	var facts []Fact
	var diagnostics []Diagnostic
	for _, scanner := range scanners {
		if scanner == nil {
			continue
		}
		result := scanner.Scan(runtime)
		facts = append(facts, result.Facts...)
		diagnostics = append(diagnostics, result.Diagnostics...)
	}
	inventory := BuildForest(facts)
	inventory.Diagnostics = append(inventory.Diagnostics, diagnostics...)
	return SortInventory(inventory)
}
