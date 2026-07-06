//go:build !linux

package zltech

import "fmt"

// NewWheelsAuto is only available on Linux builds. On other platforms, it returns an error.
func NewWheelsAuto(vendorID string, opts ...WheelsOption) (*Wheels, error) {
	return nil, fmt.Errorf("NewWheelsAuto is only supported on Linux")
}
