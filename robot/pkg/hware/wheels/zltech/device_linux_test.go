package zltech

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxFinder_MatchingVendor(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-specific test")
	}

	tmp := t.TempDir()
	netRoot := filepath.Join(tmp, "class", "net")
	deviceNet := filepath.Join(tmp, "devices", "usb1", "1-1", "1-1:1.0", "net")
	if err := os.MkdirAll(deviceNet, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(netRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "devices", "usb1", "1-1", "idVendor"), []byte("1d50\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "devices", "usb1", "1-1", "idProduct"), []byte("606f\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(deviceNet, "can2"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(deviceNet, "can2"), filepath.Join(netRoot, "can2")); err != nil {
		t.Fatal(err)
	}

	oldRoot := sysClassNetRoot
	sysClassNetRoot = netRoot
	t.Cleanup(func() { sysClassNetRoot = oldRoot })

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
