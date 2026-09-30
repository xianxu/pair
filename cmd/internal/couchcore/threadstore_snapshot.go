package couchcore

import "fmt"

func (s *ThreadStore) appendSlotSnapshots(snapshot ThreadSnapshot, manifest threadManifest) (ThreadSnapshot, error) {
	backends, err := s.discoveredBackendsFromRoots(manifest.SlotRepositories)
	if err != nil {
		return snapshot, err
	}
	kept := snapshot.Records[:0]
	for _, record := range snapshot.Records {
		local := false
		for _, backend := range backends {
			if _, belongs, _ := RecordCheckoutMembership(record, backend.slot.RepoIdentity, backend.slot.WorktreeRoot); belongs {
				local = true
				break
			}
		}
		if !local {
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
		for _, family := range manifest.RepositoryFamilies {
			if family.PrimaryRoot == backend.slot.PrimaryRoot && family.RepoIdentity == backend.slot.RepoIdentity {
				observation.StartingPath, err = ProjectFamilyPath(backend.slot.WorktreeRoot, family.RelativeStart)
				if err != nil {
					return snapshot, err
				}
				break
			}
		}
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
