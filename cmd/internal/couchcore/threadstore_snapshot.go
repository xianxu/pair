package couchcore

import "fmt"

func (s *ThreadStore) appendSlotSnapshots(snapshot ThreadSnapshot) (ThreadSnapshot, error) {
	backends, err := s.discoveredBackends()
	if err != nil {
		return snapshot, err
	}
	localPaths := map[string]bool{}
	for _, backend := range backends {
		localPaths[backend.slot.WorktreeRoot] = true
	}
	kept := snapshot.Records[:0]
	for _, record := range snapshot.Records {
		if !localPaths[record.StartingPath] {
			kept = append(kept, record)
		}
	}
	snapshot.Records = kept
	seen := map[ThreadAddress]bool{}
	for _, record := range kept {
		seen[record.Address] = true
	}
	for _, backend := range backends {
		local, err := backend.Snapshot()
		observation := SlotInventoryObservation{Identity: *backend.slot, Err: err}
		if len(local.Unreadable) != 0 {
			observation.Address = local.Unreadable[0]
			observation.Err = fmt.Errorf("slot current record unreadable: %s", backend.root)
		}
		for _, record := range local.Records {
			if seen[record.Address] {
				return snapshot, fmt.Errorf("duplicate current conversation %+v", record.Address)
			}
			seen[record.Address] = true
			observation.Address = record.Address
			snapshot.Records = append(snapshot.Records, record)
		}
		snapshot.Slots = append(snapshot.Slots, observation)
	}
	return snapshot, nil
}
