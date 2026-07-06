package thrower

import (
	"context"
	"log/slog"
)

// MockThrower returns a no-op implementation of the Thrower interface for testing purposes.
// It implements all Thrower methods but performs no actual operations, always returning nil errors.
// This is useful for testing code that depends on a Thrower without requiring actual hardware.
func MockThrower(logger *slog.Logger) Thrower {
	return &mockThrower{
		logger: logger,
	}
}

// mockThrower is a no-op implementation of the Thrower interface.
// It's used for testing scenarios where actual hardware interaction is not required.
type mockThrower struct {
	logger *slog.Logger
}

// Set is a no-op implementation that accepts throw settings but performs no actual configuration.
// It always returns nil, indicating success.
func (t *mockThrower) Set(s Settings) error {
	t.logger.Info("Set called with settings", "settings", s)
	return nil
}

// Throw is a no-op implementation that simulates a ball throw without any actual hardware interaction.
// It always returns nil, indicating success.
func (t *mockThrower) Throw(ctx context.Context) error {
	t.logger.Info("Throw called")
	return nil
}

// Load is a no-op implementation that simulates waiting for a ball to load.
// It always returns nil, indicating success.
func (t *mockThrower) Load(ctx context.Context) error {
	t.logger.Info("Load called")
	return nil
}

// Spin is a no-op implementation that simulates spinning the dispenser motor.
// It always returns nil, indicating success.
func (t *mockThrower) Spin(speed float64) error {
	t.logger.Info("Spin called with speed", "speed", speed)
	return nil
}

// Info returns a mock implementation of the Info method.
func (t *mockThrower) Info() (Info, error) {
	t.logger.Info("Info called")
	return Info{
		Loaded:         true,
		DispenserSpeed: 0,
		ThrowSettings:  Settings{},
	}, nil
}
