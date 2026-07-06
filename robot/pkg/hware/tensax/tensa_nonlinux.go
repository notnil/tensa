//go:build !linux

package tensax

import (
	"fmt"
	"time"

	"github.com/notnil/tensa/pkg/hware/wheels"
)

func (t *Tensa) setupWheels() error {
	t.logger.Info("Setting up wheels")
	var baseWheels wheels.Wheels

	switch t.c.Wheels.Type {
	case "mock":
		t.logger.Info("Using Mock Wheels")
		baseWheels = wheels.NewMock(t.logger)
	default:
		return fmt.Errorf("invalid wheels type: %s", t.c.Wheels.Type)
	}

	// Wrap with defensive timeout protection if configured
	if t.c.Wheels.CommandTimeout > 0 {
		t.logger.Info("Wrapping wheels with defensive timeout protection",
			"timeout", t.c.Wheels.CommandTimeout)
		defensiveWheels := wheels.NewDefensive(baseWheels, time.Duration(t.c.Wheels.CommandTimeout), t.logger)
		t.wheels = defensiveWheels
		t.addCloser(defensiveWheels)
	} else {
		t.wheels = baseWheels
	}

	return nil
}
