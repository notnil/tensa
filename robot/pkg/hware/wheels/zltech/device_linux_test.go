package zltech

import (
	"context"
	"runtime"
	"testing"
)

func TestLinuxFinder_NoMatch(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-specific test")
	}

	f := NewLinuxFinder("1d50")
	name, err := f.Find(context.Background())
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	expected := "can2"
	if name != expected {
		t.Fatalf("expected %s, got %s", expected, name)
	}
}
