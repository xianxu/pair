package historyarchive

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const (
	MaxIndexBytes        = 64 << 20
	MaxIndexRecords      = 100000
	MaxIndexLocations    = 16
	MaxIndexDestinations = 1024
)

// LabelIndex is discovery metadata. It never authorizes local deletion or bypasses restore verification.
// pair:m5-concept pure
type LabelIndex struct {
	Version              int                               `json:"version"`
	Records              map[string]LabelIndexRecord       `json:"records"`
	CompleteDestinations map[string]DestinationObservation `json:"complete_destinations,omitempty"`
	Extensions           map[string]json.RawMessage        `json:"-"`
}

// DestinationObservation retains completeness even when no bundle was present.
// pair:m5-concept pure
type DestinationObservation struct {
	Backend    string                     `json:"backend"`
	Locator    string                     `json:"locator"`
	FolderID   string                     `json:"folder_id"`
	ObservedAt time.Time                  `json:"observed_at"`
	Extensions map[string]json.RawMessage `json:"-"`
}

func (d DestinationObservation) Identity() string { return d.Backend + "\n" + d.Locator }

func validateDestinationObservation(d DestinationObservation) error {
	parts := strings.SplitN(d.Locator, ":", 2)
	if d.Backend != "drive" || len(parts) != 2 || !validTime(d.ObservedAt) || !validRemoteIdentity(d.FolderID) {
		return errors.New("invalid complete destination observation")
	}
	_, err := ParseDriveDestination(parts[0], parts[1])
	return err
}

// pair:m5-concept pure
type LabelIndexRecord struct {
	Label      BundleLabel                `json:"label"`
	Locations  map[string]BundleLocation  `json:"locations"`
	Extensions map[string]json.RawMessage `json:"-"`
}

// BundleLocation retains the latest observation and the latest historical verification separately.
// pair:m5-concept pure
type BundleLocation struct {
	Backend       string                     `json:"backend"`
	Locator       string                     `json:"locator"`
	FolderID      string                     `json:"folder_id"`
	BundleKey     string                     `json:"bundle_key"`
	LabelKey      string                     `json:"label_key"`
	Available     bool                       `json:"available"`
	Evidence      string                     `json:"evidence"`
	FirstObserved time.Time                  `json:"first_observed"`
	LastObserved  time.Time                  `json:"last_observed"`
	Proof         LocationProof              `json:"proof"`
	Extensions    map[string]json.RawMessage `json:"-"`
}

// pair:m5-concept pure
type LocationProof struct {
	Method     string                     `json:"method"`
	SHA256     string                     `json:"sha256"`
	VerifiedAt time.Time                  `json:"verified_at"`
	Extensions map[string]json.RawMessage `json:"-"`
}

// IndexRefresh is a complete observation of one configured logical destination.
// pair:m5-concept pure
type IndexRefresh struct {
	Backend      string
	Locator      string
	FolderID     string
	At           time.Time
	Complete     bool
	Observations []IndexObservation
}

// pair:m5-concept pure
type IndexObservation struct {
	Label    BundleLabel
	Location BundleLocation
}

func (l BundleLocation) Identity() string { return l.Backend + "\n" + l.Locator + "\n" + l.FolderID }

func validateLocation(l BundleLocation, label BundleLabel) error {
	if l.Backend != "local" && l.Backend != "drive" {
		return errors.New("unsupported location backend")
	}
	if l.Backend == "local" {
		if !filepath.IsAbs(l.Locator) || strings.ContainsRune(l.Locator, 0) || l.FolderID != "" {
			return errors.New("invalid local location")
		}
		b, k, err := BundleKeys(label.Kind, label.From, label.To, label.ID)
		if err != nil || l.BundleKey != b || l.LabelKey != k {
			return errors.New("local location keys mismatch")
		}
	} else {
		parts := strings.SplitN(l.Locator, ":", 2)
		if len(parts) != 2 {
			return errors.New("invalid Drive locator")
		}
		if _, err := ParseDriveDestination(parts[0], parts[1]); err != nil {
			return err
		}
		if l.FolderID == "" || strings.ContainsAny(l.FolderID, "\n\r\x00") || len(l.FolderID) > 1024 || l.BundleKey != label.ID+".tar.zst.gpg" || l.LabelKey != label.ID+".json" {
			return errors.New("invalid Drive location identity")
		}
	}
	if !validTime(l.FirstObserved) || !validTime(l.LastObserved) || l.LastObserved.Before(l.FirstObserved) {
		return errors.New("invalid observation times")
	}
	if l.Evidence != "observed" && l.Evidence != "verified" {
		return errors.New("invalid location evidence")
	}
	if l.Proof.Method == "" {
		if l.Proof.SHA256 != "" || !l.Proof.VerifiedAt.IsZero() || l.Evidence == "verified" {
			return errors.New("missing verification proof")
		}
	} else {
		if l.Proof.Method != "backend-sha256" && l.Proof.Method != "upload-receipt" && l.Proof.Method != "local-catalog" {
			return errors.New("unsupported verification proof")
		}
		if l.Backend == "drive" && l.Proof.Method == "local-catalog" {
			return errors.New("local proof cannot verify Drive")
		}
		if l.Proof.SHA256 != label.SHA256 || !validTime(l.Proof.VerifiedAt) || l.Proof.VerifiedAt.After(l.LastObserved) {
			return errors.New("invalid verification proof")
		}
	}
	return nil
}

func ValidateLabelIndex(i LabelIndex) error {
	if len(i.CompleteDestinations) > MaxIndexDestinations {
		return errors.New("complete destination count exceeds 1024")
	}
	for key, d := range i.CompleteDestinations {
		if key != d.Identity() {
			return errors.New("complete destination identity mismatch")
		}
		if err := validateDestinationObservation(d); err != nil {
			return err
		}
	}
	if i.Version != Version || i.Records == nil || len(i.Records) > MaxIndexRecords {
		return errors.New("invalid label index version or record bound")
	}
	for id, r := range i.Records {
		if id != r.Label.ID || ValidateLabel(r.Label) != nil || r.Locations == nil || len(r.Locations) > MaxIndexLocations {
			return errors.New("invalid index record")
		}
		for key, l := range r.Locations {
			if key != l.Identity() {
				return errors.New("location identity mismatch")
			}
			if err := validateLocation(l, r.Label); err != nil {
				return err
			}
			if marker, ok := i.CompleteDestinations[l.Backend+"\n"+l.Locator]; ok && l.Available && (l.FolderID != marker.FolderID || l.LastObserved.Before(marker.ObservedAt)) {
				return errors.New("available location contradicts complete destination observation")
			}
		}
		raw, err := json.Marshal(r)
		if err != nil {
			return err
		}
		if len(raw) > MaxMetadataBytes {
			return errors.New("index record exceeds 4 MiB")
		}
	}
	raw, err := json.Marshal(i)
	if err != nil {
		return err
	}
	if len(raw) > MaxIndexBytes {
		return errors.New("label index exceeds 64 MiB")
	}
	return nil
}

// reconcileReceiptLocation orders a dated partial attestation against complete
// destination knowledge and the existing location's last observation.
func reconcileReceiptLocation(prior BundleLocation, exists bool, marker DestinationObservation, complete bool, receipt BundleLocation) (BundleLocation, error) {
	when := receipt.Proof.VerifiedAt
	if exists && (!when.After(prior.LastObserved) || !when.After(prior.Proof.VerifiedAt)) {
		return prior, nil
	}
	if complete && (receipt.FolderID != marker.FolderID || !when.After(marker.ObservedAt)) {
		receipt.Available = false
	}
	if !exists {
		return receipt, nil
	}
	if prior.FirstObserved.Before(receipt.FirstObserved) {
		receipt.FirstObserved = prior.FirstObserved
	}
	var err error
	receipt.Extensions, err = mergeExtensions(prior.Extensions, receipt.Extensions)
	if err != nil {
		return BundleLocation{}, err
	}
	receipt.Proof.Extensions, err = mergeExtensions(prior.Proof.Extensions, receipt.Proof.Extensions)
	if err != nil {
		return BundleLocation{}, err
	}
	return receipt, nil
}

func cloneIndex(i LabelIndex) (LabelIndex, error) {
	raw, err := json.Marshal(i)
	if err != nil {
		return LabelIndex{}, err
	}
	return DecodeLabelIndex(raw)
}

func mergeExtensions(a, b map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	out := make(map[string]json.RawMessage, len(a)+len(b))
	for k, v := range a {
		out[k] = append(json.RawMessage(nil), v...)
	}
	for k, v := range b {
		if prior, ok := out[k]; ok {
			var x, y any
			dx, dy := json.NewDecoder(bytes.NewReader(prior)), json.NewDecoder(bytes.NewReader(v))
			dx.UseNumber()
			dy.UseNumber()
			if dx.Decode(&x) != nil || dy.Decode(&y) != nil {
				return nil, errors.New("invalid extension JSON")
			}
			bx, _ := json.Marshal(x)
			by, _ := json.Marshal(y)
			if !bytes.Equal(bx, by) {
				return nil, fmt.Errorf("conflicting extension %q", k)
			}
		}
		out[k] = append(json.RawMessage(nil), v...)
	}
	return out, nil
}

// ApplyIndexRefresh preserves its input and commits no partial observation.
func ApplyIndexRefresh(old LabelIndex, r IndexRefresh) (LabelIndex, error) {
	if !r.Complete || !validTime(r.At) {
		return LabelIndex{}, errors.New("refresh is incomplete or has invalid time")
	}
	if r.Backend != "local" && r.Backend != "drive" {
		return LabelIndex{}, errors.New("unsupported refresh backend")
	}
	if err := ValidateLabelIndex(old); err != nil {
		return LabelIndex{}, err
	}
	out, err := cloneIndex(old)
	if err != nil {
		return LabelIndex{}, err
	}
	if r.Backend == "drive" {
		marker := DestinationObservation{Backend: r.Backend, Locator: r.Locator, FolderID: r.FolderID, ObservedAt: r.At}
		if err := validateDestinationObservation(marker); err != nil {
			return LabelIndex{}, err
		}
		if prior, ok := out.CompleteDestinations[marker.Identity()]; ok {
			if r.At.Before(prior.ObservedAt) {
				return LabelIndex{}, errors.New("stale complete destination refresh")
			}
			marker.Extensions = prior.Extensions
		}
		if out.CompleteDestinations == nil {
			out.CompleteDestinations = map[string]DestinationObservation{}
		}
		out.CompleteDestinations[marker.Identity()] = marker
	}
	for id, record := range out.Records {
		for key, loc := range record.Locations {
			if loc.Backend == r.Backend && loc.Locator == r.Locator {
				if r.At.Before(loc.LastObserved) {
					return LabelIndex{}, errors.New("stale index refresh")
				}
				loc.Available = false
				record.Locations[key] = loc
			}
		}
		out.Records[id] = record
	}
	seen := map[string]bool{}
	for _, ob := range r.Observations {
		l := ob.Location
		if r.Backend == "drive" && l.FolderID != r.FolderID {
			return LabelIndex{}, errors.New("observation folder differs from complete binding")
		}
		if l.Backend != r.Backend || l.Locator != r.Locator || (l.Available && !l.LastObserved.Equal(r.At)) || (!l.Available && (r.Backend != "local" || l.LastObserved.After(r.At))) {
			return LabelIndex{}, errors.New("observation does not match refresh")
		}
		if err := ValidateLabel(ob.Label); err != nil {
			return LabelIndex{}, err
		}
		if err := validateLocation(l, ob.Label); err != nil {
			return LabelIndex{}, err
		}
		id := ob.Label.ID
		if seen[id] {
			return LabelIndex{}, errors.New("duplicate bundle in refresh")
		}
		seen[id] = true
		record, ok := out.Records[id]
		if !ok {
			record = LabelIndexRecord{Label: ob.Label, Locations: map[string]BundleLocation{}}
		} else {
			a, b := record.Label, ob.Label
			a.Extensions = nil
			b.Extensions = nil
			ra, _ := json.Marshal(a)
			rb, _ := json.Marshal(b)
			if !bytes.Equal(ra, rb) {
				return LabelIndex{}, errors.New("conflicting label for bundle ID")
			}
			record.Label.Extensions, err = mergeExtensions(record.Label.Extensions, ob.Label.Extensions)
			if err != nil {
				return LabelIndex{}, err
			}
		}
		if prior, ok := record.Locations[l.Identity()]; ok {
			if !l.Available && prior.LastObserved.After(l.LastObserved) {
				l.LastObserved = prior.LastObserved
			}
			if l.FirstObserved.After(prior.FirstObserved) {
				l.FirstObserved = prior.FirstObserved
			}
			l.Extensions, err = mergeExtensions(prior.Extensions, l.Extensions)
			if err != nil {
				return LabelIndex{}, err
			}
			l.Proof.Extensions, err = mergeExtensions(prior.Proof.Extensions, l.Proof.Extensions)
			if err != nil {
				return LabelIndex{}, err
			}
			if l.Proof.Method == "" || prior.Proof.VerifiedAt.After(l.Proof.VerifiedAt) {
				ext := l.Proof.Extensions
				l.Proof = prior.Proof
				l.Proof.Extensions = ext
			}
		}
		record.Locations[l.Identity()] = l
		out.Records[id] = record
	}
	if err := ValidateLabelIndex(out); err != nil {
		return LabelIndex{}, err
	}
	return out, nil
}
