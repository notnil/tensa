package player

import (
	"github.com/notnil/tensa/pkg/ai/location"
)

// Provider provides access to current player locations.
type Provider interface {
	Players() ([]location.Loc, error)
}
