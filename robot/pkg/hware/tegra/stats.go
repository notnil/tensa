package tegra

import "time"

// UsageValueUnit stores a used/total value and its associated unit.
type UsageValueUnit struct {
	Used  float64 `json:"used"`
	Total float64 `json:"total"`
}

// Stats holds the parsed information from the tegrastats output.
// Fields are pointers to allow for missing values in different tegrastats versions or device types.
type Stats struct {
	Timestamp     time.Time          `json:"timestamp"`                // Time of the reading
	RAM_MB        UsageValueUnit     `json:"ram_mb"`                   // RAM usage (e.g., 1234/8192MB)
	Swap_MB       UsageValueUnit     `json:"swap_mb"`                  // Swap usage (e.g., 0/4096MB)
	GPUUtil       float64            `json:"gpu_util"`                 // GPU utilization (e.g., 50%)
	CPUUtil       map[string]float64 `json:"cpu_util,omitempty"`       // CPU utilization per core (e.g., [10%@1000, OFF, 5%@1000, OFF])
	TemperaturesC map[string]float64 `json:"temperatures_c,omitempty"` // Temperatures (e.g., {"CPU": 45.5C, "GPU": 48.0C})
	PowerMW       map[string]float64 `json:"power_mw,omitempty"`       // Power consumption (e.g., {"CPU": 1500mW, "GPU": 3000mW})
}

type Parser interface {
	Parse(line string) (Stats, error)
}
