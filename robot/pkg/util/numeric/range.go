package numeric

import (
	"encoding/json"
	"math/rand"

	"golang.org/x/exp/constraints"
)

type Rangable interface {
	constraints.Integer | constraints.Float
}

// Range represents a range of numeric values
type Range[T Rangable] struct {
	Min, Max T
}

// NewRange creates a new range
func NewRange[T Rangable](min, max T) Range[T] {
	if min > max {
		panic("min cannot be greater than max")
	}
	return Range[T]{Min: min, Max: max}
}

// In checks if a value is within the range
// Returns true if the value is within the range, false otherwise
// The value is considered within the range if it is equal to the min or max values
func (r Range[T]) In(value T) bool {
	return value >= r.Min && value <= r.Max
}

// Midde returns the middle value of the range
func (r Range[T]) Middle() T {
	return r.Min + (r.Max-r.Min)/2
}

// Sample returns a random value within the range
func (r Range[T]) Sample(rd *rand.Rand) T {
	if r.Min == r.Max {
		return r.Min
	}

	switch any(r.Min).(type) {
	case int:
		return T(rd.Intn(int(r.Max-r.Min+1)) + int(r.Min))
	case int8:
		return T(rd.Intn(int(r.Max-r.Min+1)) + int(r.Min))
	case int16:
		return T(rd.Intn(int(r.Max-r.Min+1)) + int(r.Min))
	case int32:
		return T(rd.Int31n(int32(r.Max-r.Min+1)) + int32(r.Min))
	case int64:
		return T(rd.Int63n(int64(r.Max-r.Min+1)) + int64(r.Min))
	case uint:
		return T(rd.Intn(int(r.Max-r.Min+1)) + int(r.Min))
	case uint8:
		return T(rd.Intn(int(r.Max-r.Min+1)) + int(r.Min))
	case uint16:
		return T(rd.Intn(int(r.Max-r.Min+1)) + int(r.Min))
	case uint32:
		return T(rd.Uint32()%(uint32(r.Max-r.Min+1)) + uint32(r.Min))
	case uint64:
		return T(rd.Uint64()%(uint64(r.Max-r.Min+1)) + uint64(r.Min))
	case float32:
		return T(rd.Float32()*(float32(r.Max)-float32(r.Min)) + float32(r.Min))
	case float64:
		return T(rd.Float64()*(float64(r.Max)-float64(r.Min)) + float64(r.Min))
	default:
		panic("unsupported type")
	}
}

// Clamp returns the value bound to the range
func (r Range[T]) Clamp(v T) T {
	if v < r.Min {
		return r.Min
	}
	if v > r.Max {
		return r.Max
	}
	return v
}

// MarshalJSON implements the json.Marshaler interface
func (r Range[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Min T `json:"min"`
		Max T `json:"max"`
	}{
		Min: r.Min,
		Max: r.Max,
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (r *Range[T]) UnmarshalJSON(data []byte) error {
	var aux struct {
		Min T `json:"min"`
		Max T `json:"max"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	r.Min = aux.Min
	r.Max = aux.Max
	return nil
}
