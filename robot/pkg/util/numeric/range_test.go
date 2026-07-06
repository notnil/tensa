package numeric

import (
	"encoding/json"
	"math/rand"
	"testing"
	"time"
)

func TestRange_In(t *testing.T) {
	tests := []struct {
		name  string
		r     Range[int]
		value int
		want  bool
	}{
		{"Value in range", Range[int]{Min: 1, Max: 10}, 5, true},
		{"Value equal to Min", Range[int]{Min: 1, Max: 10}, 1, true},
		{"Value equal to Max", Range[int]{Min: 1, Max: 10}, 10, true},
		{"Value below range", Range[int]{Min: 1, Max: 10}, 0, false},
		{"Value above range", Range[int]{Min: 1, Max: 10}, 11, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.In(tt.value); got != tt.want {
				t.Errorf("Range.In() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRange_Sample(t *testing.T) {
	tests := []struct {
		name string
		r    Range[int]
	}{
		{"Sample in range 1-10", Range[int]{Min: 1, Max: 10}},
		{"Sample in range 10-20", Range[int]{Min: 10, Max: 20}},
	}

	rd := rand.New(rand.NewSource(time.Now().UnixNano()))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.Sample(rd)
			if !tt.r.In(got) {
				t.Errorf("Range.Sample() = %v, not in range [%v, %v]", got, tt.r.Min, tt.r.Max)
			}
		})
	}
}

func TestRange_JSON(t *testing.T) {
	tests := []struct {
		name string
		r    Range[int]
	}{
		{"Range 1-10", Range[int]{Min: 1, Max: 10}},
		{"Range -5 to 5", Range[int]{Min: -5, Max: 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.r)
			if err != nil {
				t.Fatalf("Error marshaling to JSON: %v", err)
			}

			var decodedRange Range[int]
			if err := json.Unmarshal(data, &decodedRange); err != nil {
				t.Fatalf("Error unmarshaling from JSON: %v", err)
			}

			if decodedRange != tt.r {
				t.Errorf("Decoded range = %+v, want %+v", decodedRange, tt.r)
			}
		})
	}
}
