package historyarchive

import (
	"errors"
	"sort"
	"time"
)

// FindQuery selects public labels by kind and UTC interval overlap.
// pair:m5-concept pure
type FindQuery struct {
	Kind  string
	Since time.Time
	Until time.Time
}

func ParseFindTime(raw string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02", time.RFC3339Nano} {
		if t, err := time.Parse(layout, raw); err == nil {
			if !validTime(t) {
				return time.Time{}, errors.New("time bound must be a nonzero UTC date or RFC3339 timestamp")
			}
			return t.UTC(), nil
		}
	}
	return time.Time{}, errors.New("time bound must be YYYY-MM-DD or a UTC RFC3339 timestamp")
}

func SelectLabels(index LabelIndex, q FindQuery) ([]LabelIndexRecord, error) {
	if err := ValidateLabelIndex(index); err != nil {
		return nil, err
	}
	if q.Kind != "" {
		if err := validateKind(q.Kind); err != nil {
			return nil, err
		}
	}
	if (!q.Since.IsZero() && !validTime(q.Since)) || (!q.Until.IsZero() && !validTime(q.Until)) || (!q.Since.IsZero() && !q.Until.IsZero() && !q.Since.Before(q.Until)) {
		return nil, errors.New("invalid UTC find interval")
	}
	out := make([]LabelIndexRecord, 0)
	for _, r := range index.Records {
		l := r.Label
		if q.Kind != "" && l.Kind != q.Kind {
			continue
		}
		if !q.Since.IsZero() && l.To.Before(q.Since) {
			continue
		}
		if !q.Until.IsZero() && !l.From.Before(q.Until) {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(a, b int) bool {
		x, y := out[a].Label, out[b].Label
		if x.Kind != y.Kind {
			return x.Kind < y.Kind
		}
		if !x.From.Equal(y.From) {
			return x.From.Before(y.From)
		}
		return x.ID < y.ID
	})
	return out, nil
}
