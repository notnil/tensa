//go:build linux

package zltech

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/notnil/canbus"
)

const DefaultSyncPeriod = 5 * time.Millisecond

// NewWheelsAuto detects the CAN interface by USB vendor ID, constructs clients for
// node IDs 1 and 2 on that interface, and delegates to NewWheels.
func NewWheelsAuto(vendorID string, syncPeriod time.Duration, logger *slog.Logger, opts ...WheelsOption) (*Wheels, error) {
	ctx := context.Background()

	finder := NewLinuxFinder(vendorID)
	ifaceName, err := finder.Find(ctx)
	if err != nil {
		return nil, fmt.Errorf("detect CAN interface: %w", err)
	}

	logger.Info("detected CAN interface", slog.String("interface", ifaceName))

	// Declare variables before any goto to avoid jump-over-declaration errors
	var (
		up         bool
		bitrate    = uint32(Bitrate)
		restartMs  = uint32(RestartMs)
		txQueueLen = int(TxQueueLen)
	)

	// Try to check if interface is up, but don't fail if we can't
	// This may fail in containers even with host networking due to netlink restrictions
	up, err = canbus.IsInterfaceUp(ifaceName)
	if err != nil {
		logger.Warn("cannot check interface status via netlink, will attempt direct connection",
			slog.String("interface", ifaceName),
			slog.Any("error", err))
		// Skip interface configuration and try direct connection
		goto connectBus
	}

	// If we successfully checked status, try to configure the interface
	if up {
		if err := canbus.SetInterfaceDown(ifaceName); err != nil {
			logger.Warn("cannot set interface down",
				slog.String("interface", ifaceName),
				slog.Any("error", err))
			goto connectBus
		}
	}
	if err := canbus.ConfigureLinuxCANInterface(ifaceName, canbus.LinuxCANInterfaceOptions{
		Bitrate:    &bitrate,
		RestartMs:  &restartMs,
		TxQueueLen: &txQueueLen,
	}); err != nil {
		logger.Warn("cannot configure interface, assuming already configured",
			slog.String("interface", ifaceName),
			slog.Any("error", err))
		goto connectBus
	}

	if err := canbus.SetInterfaceUp(ifaceName); err != nil {
		logger.Warn("cannot set interface up, will try to connect anyway",
			slog.String("interface", ifaceName),
			slog.Any("error", err))
	}

connectBus:
	// The critical part - open the SocketCAN connection
	// This will fail if the interface truly isn't usable
	boolPtr := func(b bool) *bool { return &b }
	bus, err := canbus.DialSocketCANWithOptions(ifaceName, &canbus.SocketCANOptions{
		Loopback:           boolPtr(false),
		ReceiveOwnMessages: boolPtr(false),
		SendBufferBytes:    1 << 20,
		ReceiveBufferBytes: 1 << 20,
	})
	if err != nil {
		return nil, fmt.Errorf("open CAN bus %s: %w", ifaceName, err)
	}
	mux := canbus.NewMux(bus)
	c1 := NewWithMux(bus, mux, NodeID1, WithVelocityViaRPDO(), WithSyncPeriod(syncPeriod), WithLogger(logger))
	c2 := NewWithMux(bus, mux, NodeID2, WithVelocityViaRPDO(), WithSyncPeriod(syncPeriod), WithLogger(logger))
	opts = append(opts, WithWheelsLogger(logger))
	w, wErr := NewWheels(c1, c2, opts...)
	if wErr != nil {
		_ = c1.Close()
		_ = c2.Close()
		_ = bus.Close()
		return nil, wErr
	}
	return w, nil
}
