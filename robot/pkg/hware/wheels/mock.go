package wheels

import "log/slog"

// Mock is a no-op implementation of the Wheels interface for testing and development.
// It logs all method calls but performs no actual hardware operations.
var _ Wheels = &Mock{}

// Mock implements the Wheels interface with no-op operations.
type Mock struct {
	logger *slog.Logger // Logger for recording mock operations
}

// NewMock creates a new Mock wheels instance with the given logger.
func NewMock(logger *slog.Logger) *Mock {
	return &Mock{logger: logger}
}

// Move simulates a wheel movement command, logging the direction and speed.
func (m *Mock) Move(dir, speed float64) error {
	m.logger.Info("Wheels Mock: Move", "dir", dir, "speed", speed)
	return nil
}

// Rotate simulates a rotation command, logging the rotation speed.
func (m *Mock) Rotate(speed float64) error {
	m.logger.Info("Wheels Mock: Rotate", "speed", speed)
	return nil
}

// Status returns an empty Status struct, simulating a status check.
func (m *Mock) Status() (Status, error) {
	m.logger.Info("Wheels Mock: Status")
	return Status{}, nil
}

// Stop simulates stopping all wheels, logging the operation.
func (m *Mock) Stop() error {
	m.logger.Info("Wheels Mock: Stop")
	return nil
}

// Disable simulates disabling wheel control, logging the operation.
func (m *Mock) Disable() error {
	m.logger.Info("Wheels Mock: Disable")
	return nil
}

// Enable simulates enabling wheel control, logging the operation.
func (m *Mock) Enable() error {
	m.logger.Info("Wheels Mock: Enable")
	return nil
}
