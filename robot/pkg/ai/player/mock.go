package player

import (
	"github.com/notnil/tensa/pkg/ai/location"
)

// MockProvider is a mock implementation of the Provider interface for testing.
type MockProvider struct {
	players []location.Loc
}

// NewMockProvider creates a new mock player provider with no players.
func NewMockProvider() *MockProvider {
	return &MockProvider{
		players: []location.Loc{},
	}
}

// Players returns the mock player locations.
func (m *MockProvider) Players() ([]location.Loc, error) {
	return m.players, nil
}

// SetPlayers sets the mock player locations for testing.
func (m *MockProvider) SetPlayers(players []location.Loc) {
	m.players = players
}
