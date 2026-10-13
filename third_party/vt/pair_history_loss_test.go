package vt

import (
	uv "github.com/charmbracelet/ultraviolet"
	"testing"
)

func TestHistoryCapacityLossSurvivesClear(t *testing.T) {
	for _, kind := range []string{"rows", "cells", "bytes", "oversized-row", "oversized-byte-row"} {
		t.Run(kind, func(t *testing.T) {
			limits := DefaultLimits()
			limits.HistoryLines, limits.HistoryCells, limits.HistoryBytes = 100, 1000, 1<<20
			raw := "aaaa\r\nbbbb\r\ncccc\r\n"
			switch kind {
			case "rows":
				limits.HistoryLines = 1
			case "cells":
				limits.HistoryCells = 4
			case "bytes":
				limits.HistoryBytes = lineBytes(uv.Line{{Content: "a", Width: 1}, {Content: "a", Width: 1}, {Content: "a", Width: 1}, {Content: "a", Width: 1}})
			case "oversized-byte-row":
				limits.HistoryBytes = 1
			case "oversized-row":
				limits.HistoryCells = 3
			}
			em, err := NewEmulatorWithLimits(10, 1, limits)
			if err != nil {
				t.Fatal(err)
			}
			em.Write([]byte(raw))
			lost := em.HistoryCapacityLoss()
			if lost == 0 {
				t.Fatal("capacity loss was not reported")
			}
			em.ClearScrollback()
			if em.HistoryCapacityLoss() != lost {
				t.Fatal("clear erased capacity loss")
			}
		})
	}
}

func TestHistoryClearIsNotCapacityLoss(t *testing.T) {
	em := NewEmulator(10, 1)
	em.Write([]byte("one\r\ntwo\r\n\x1b[3Jafter\r\n"))
	if got := em.HistoryCapacityLoss(); got != 0 {
		t.Fatalf("intentional clear counted as %d lost rows", got)
	}
}
