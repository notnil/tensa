package zltech

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Finder interface {
	Find(ctx context.Context) (string, error)
}

// LinuxFinder implements Finder by resolving CAN interfaces in sysfs
// and matching their parent USB device vendor against a provided VID.
// It returns the first and only matching interface name (e.g., "can0").
// If none or multiple match, an error is returned.
type LinuxFinder struct {
	vendorID string
}

// sysClassNetRoot allows tests to override the sysfs root used for discovery.
var sysClassNetRoot = "/sys/class/net"

// NewLinuxFinder creates a new LinuxFinder that searches for a CAN interface
// whose parent USB device has the given vendorID (hex, e.g., "1d50" or "0x1d50").
func NewLinuxFinder(vendorID string) *LinuxFinder {
	vid := normalizeVID(vendorID)
	return &LinuxFinder{vendorID: vid}
}

// Find implements Finder. It enumerates /sys/class/net/can* and checks the
// parent USB device's idVendor. Returns interface name (canX) on unique match.
func (f *LinuxFinder) Find(ctx context.Context) (string, error) {
	if f.vendorID == "" {
		return "", fmt.Errorf("empty vendor id")
	}

	paths, err := filepath.Glob(filepath.Join(sysClassNetRoot, "can*"))
	if err != nil {
		return "", fmt.Errorf("glob can interfaces: %w", err)
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("no CAN interfaces found under /sys/class/net")
	}

	var matches []string
	for _, p := range paths {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		name := filepath.Base(p)
		// Resolve symlink to the device path, then walk up to find a USB node with idVendor
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil {
			// Skip interfaces we cannot resolve
			continue
		}
		// Typical layout: /sys/class/net/canX -> .../usbX/Y/…/Y:1.0/net/canX
		usbDevDir, err := findUSBDeviceDir(resolved)
		if err != nil {
			// Skip non-USB or interfaces without id files
			continue
		}
		vidBytes, err := os.ReadFile(filepath.Join(usbDevDir, "idVendor"))
		if err != nil {
			// Skip if vendor cannot be read
			continue
		}
		vid := normalizeVID(string(vidBytes))
		if vid == f.vendorID {
			matches = append(matches, name)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no CAN interfaces matched vendor id %s", f.vendorID)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple CAN interfaces matched vendor id %s: %s", f.vendorID, strings.Join(matches, ", "))
	}
}

// findUSBDeviceDir walks up from a resolved sysfs network path to locate the
// nearest parent directory that represents a USB device (has idVendor/idProduct).
func findUSBDeviceDir(start string) (string, error) {
	// Walk up directories until root, looking for idVendor/idProduct files
	dir := start
	for {
		if fileExists(filepath.Join(dir, "idVendor")) && fileExists(filepath.Join(dir, "idProduct")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir || parent == "/" || parent == "." {
			break
		}
		dir = parent
	}
	return "", errors.New("no usb id files found in parent chain")
}

func fileExists(p string) bool {
	if _, err := os.Stat(p); err == nil {
		return true
	}
	return false
}

// normalizeVID lowercases, trims, and strips optional 0x prefix and newlines.
// Returns empty string if input is empty after normalization.
func normalizeVID(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimPrefix(s, "0x")
	return s
}
