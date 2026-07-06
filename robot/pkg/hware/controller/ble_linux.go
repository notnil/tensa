//go:build linux

package controller

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/linux/cmd"
)

const (
	DefaultDeviceID = 0
)

var DefaultServerOptions = []gatt.Option{
	gatt.LnxMaxConnections(1),
	gatt.LnxDeviceID(DefaultDeviceID, true),
	gatt.LnxSetAdvertisingParameters(&cmd.LESetAdvertisingParameters{
		AdvertisingIntervalMin: 0x00f4,
		AdvertisingIntervalMax: 0x00f4,
		AdvertisingChannelMap:  0x7,
	}),
}

// ServerOptionsForMAC returns GATT server options using the adapter index
// resolved from the provided BD Address (MAC) via hciconfig.
func ServerOptionsForMAC(bdAddr string) ([]gatt.Option, error) {
	idx, err := HCIIndexFromBDAddress(bdAddr)
	if err != nil {
		return nil, err
	}
	return []gatt.Option{
		gatt.LnxMaxConnections(1),
		gatt.LnxDeviceID(idx, true),
		gatt.LnxSetAdvertisingParameters(&cmd.LESetAdvertisingParameters{
			AdvertisingIntervalMin: 0x00f4,
			AdvertisingIntervalMax: 0x00f4,
			AdvertisingChannelMap:  0x7,
		}),
	}, nil
}

// HCIIndexFromBDAddress returns the current hci adapter index for the given
// Bluetooth Device Address (MAC). It invokes `hciconfig -a` and parses the
// output to locate the adapter block that contains the BD Address.
// No filesystem fallback is performed.
//
// Example: "58:02:05:BD:AB:5C" -> 0 (for hci0)
func HCIIndexFromBDAddress(bdAddr string) (int, error) {
	if strings.TrimSpace(bdAddr) == "" {
		return -1, fmt.Errorf("bdAddr is empty")
	}

	normalizedTarget := normalizeBDAddr(bdAddr)

	out, err := exec.Command("hciconfig", "-a").Output()
	if err != nil {
		return -1, fmt.Errorf("hciconfig -a: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	headerRe := regexp.MustCompile(`^hci([0-9]+):`) // anchored
	addrRe := regexp.MustCompile(`BD\s+Address:\s*([0-9A-Fa-f:]+)`)

	currentIndex := -1
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		if m := headerRe.FindStringSubmatch(line); m != nil {
			idx, parseErr := strconv.Atoi(m[1])
			if parseErr == nil {
				currentIndex = idx
			} else {
				currentIndex = -1
			}
			continue
		}

		if currentIndex >= 0 {
			if m := addrRe.FindStringSubmatch(line); m != nil {
				addr := m[1]
				if normalizeBDAddr(addr) == normalizedTarget {
					return currentIndex, nil
				}
			}
		}
	}

	return -1, fmt.Errorf("no hci adapter found for BD Address %q", bdAddr)
}

func normalizeBDAddr(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToUpper(s)
	s = strings.ReplaceAll(s, "-", ":")
	return s
}
