//go:build !linux

package controller

import "github.com/paypal/gatt"

var DefaultServerOptions = []gatt.Option{}

// ServerOptionsForMAC is a no-op on non-Linux platforms; returns defaults.
func ServerOptionsForMAC(bdAddr string) ([]gatt.Option, error) {
	return DefaultServerOptions, nil
}
