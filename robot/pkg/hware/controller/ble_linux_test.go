//go:build linux

package controller

import (
	"log/slog"
	"testing"
)

func TestHCIIndexFromBDAddress_FoundOrSkip(t *testing.T) {
	addr := "EC:75:0C:A5:79:9E"
	wantIdx := 1
	gotIdx, err := HCIIndexFromBDAddress(addr)
	if err != nil {
		t.Fatalf("HCIIndexFromBDAddress returned error for existing addr %q: %v", addr, err)
	}
	slog.Info("resolved index from helper", "addr", addr, "gotIdx", gotIdx)
	if gotIdx != 1 {
		t.Fatalf("index mismatch for %q: got %d, want %d", addr, gotIdx, wantIdx)
	}
}
