package couchcore

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type ResourcePolicy struct {
	Cadence             time.Duration
	WriteBytesPerSecond float64
	WriteDuration       time.Duration
	ProcessRSSBytes     uint64
	IdleCPUPercent      float64
	IdleCPUDuration     time.Duration
	OrphanRSSBytes      uint64
	OrphanSamples       int
}

func ParseResourcePolicy(env map[string]string) (ResourcePolicy, error) {
	p := ResourcePolicy{time.Minute, 2 << 20, 10 * time.Minute, 8 << 30, 90, 15 * time.Minute, 256 << 20, 2}
	for k, v := range env {
		if !strings.HasPrefix(k, "COUCH_RESOURCE_") {
			continue
		}
		var err error
		switch k {
		case "COUCH_RESOURCE_CADENCE", "COUCH_RESOURCE_WRITE_DURATION", "COUCH_RESOURCE_IDLE_CPU_DURATION":
			var d time.Duration
			d, err = time.ParseDuration(v)
			if err == nil && (d < time.Second || d > 24*time.Hour) {
				err = fmt.Errorf("duration outside 1s..24h")
			}
			switch k {
			case "COUCH_RESOURCE_CADENCE":
				if d > time.Hour {
					err = fmt.Errorf("cadence exceeds 1h")
				}
				p.Cadence = d
			case "COUCH_RESOURCE_WRITE_DURATION":
				p.WriteDuration = d
			default:
				p.IdleCPUDuration = d
			}
		case "COUCH_RESOURCE_PROCESS_RSS_BYTES", "COUCH_RESOURCE_ORPHAN_RSS_BYTES":
			var n uint64
			n, err = strconv.ParseUint(v, 10, 64)
			if n == 0 || n > 1<<50 {
				err = fmt.Errorf("bytes outside 1..2^50")
			}
			if k == "COUCH_RESOURCE_PROCESS_RSS_BYTES" {
				p.ProcessRSSBytes = n
			} else {
				p.OrphanRSSBytes = n
			}
		case "COUCH_RESOURCE_WRITE_BYTES_PER_SECOND", "COUCH_RESOURCE_IDLE_CPU_PERCENT":
			var n float64
			n, err = strconv.ParseFloat(v, 64)
			if math.IsNaN(n) || math.IsInf(n, 0) || n <= 0 || n > 1e12 {
				err = fmt.Errorf("invalid threshold")
			}
			if k == "COUCH_RESOURCE_IDLE_CPU_PERCENT" {
				p.IdleCPUPercent = n
			} else {
				p.WriteBytesPerSecond = n
			}
		default:
			err = fmt.Errorf("unknown setting")
		}
		if err != nil {
			return ResourcePolicy{}, fmt.Errorf("%s: %w", k, err)
		}
	}
	return p, nil
}
