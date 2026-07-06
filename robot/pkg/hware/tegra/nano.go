package tegra

import (
	"fmt"
	"strings"
	"time"
)

const NanoExample = `05-05-2025 06:05:39 RAM 2639/7620MB (lfb 9x4MB) SWAP 0/3810MB (cached 0MB) CPU [1%@729,4%@729,8%@729,4%@729,5%@729,4%@729] GR3D_FREQ 0% cpu@44.156C soc2@43.781C soc0@44.687C gpu@45.968C tj@45.968C soc1@45.312C VDD_IN 5263mW/5263mW VDD_CPU_GPU_CV 717mW/717mW VDD_SOC 1794mW/1794mW`

const (
	NanoTempCPU  = "cpu"
	NanoTempGPU  = "gpu"
	NanoTempSOC0 = "soc0"
	NanoTempSOC1 = "soc1"
	NanoTempSOC2 = "soc2"
	NanoTempTJ   = "tj"
)

// NanoParser implements Parser for Nano tegrastats output.
type NanoParser struct{}

// Parse takes a single tegrastats line from an Nano device and returns a populated Stats.
func (p NanoParser) Parse(line string) (Stats, error) {
	var s Stats
	s.TemperaturesC = make(map[string]float64)
	s.PowerMW = make(map[string]float64)

	fields := strings.Fields(line)
	if len(fields) < 3 {
		return s, fmt.Errorf("invalid tegrastats line: %q", line)
	}
	// Timestamp is the first two fields
	var err error
	s.Timestamp, err = time.Parse("01-02-2006 15:04:05", fields[0]+" "+fields[1])
	if err != nil {
		return s, fmt.Errorf("failed to parse timestamp: %w", err)
	}
	i := 2

	// RAM usage
	if fields[i] == "RAM" && i+1 < len(fields) {
		if uv, err := parseUsage(fields[i+1]); err == nil {
			s.RAM_MB = uv
		}
		i += 2
	}
	// lfb free blocks
	if i+1 < len(fields) && strings.HasPrefix(fields[i], "(lfb") {
		// key := strings.TrimPrefix(fields[i], "(")
		// val := strings.TrimSuffix(fields[i+1], ")")
		// s.OtherMetrics[key] = val
		i += 2
	}
	// SWAP usage
	if fields[i] == "SWAP" && i+1 < len(fields) {
		if uv, err := parseUsage(fields[i+1]); err == nil {
			s.Swap_MB = uv
		}
		i += 2
	}
	// cached swap
	if i+1 < len(fields) && strings.HasPrefix(fields[i], "(cached") {
		// key := strings.TrimPrefix(fields[i], "(")
		// val := strings.TrimSuffix(fields[i+1], ")")
		// s.OtherMetrics[key] = val
		i += 2
	}
	// CPU utilizations
	s.CPUUtil = make(map[string]float64)
	if fields[i] == "CPU" && i+1 < len(fields) {
		list := strings.Trim(fields[i+1], "[]")
		items := strings.Split(list, ",")
		for idx, it := range items {
			it = strings.TrimSpace(it)
			if it == "OFF" {
				s.CPUUtil[fmt.Sprintf("core%d", idx)] = 0
			} else if parts := strings.SplitN(it, "@", 2); len(parts) == 2 {
				// Assuming the format is "percentage%@frequency" or just "percentage%"
				if v, _, err := parseValueUnit(parts[0]); err == nil {
					// Note: The Orin parser extracted frequency here, but Nano only shows %.
					// Store the percentage value and unit. Frequency might be ignored or stored elsewhere if needed.
					s.CPUUtil[fmt.Sprintf("core%d", idx)] = v
				}
			}
		}
		i += 2
	}
	// GPU utilization (GR3D_FREQ)
	if fields[i] == "GR3D_FREQ" && i+1 < len(fields) {
		if v, _, err := parseValueUnit(fields[i+1]); err == nil {
			s.GPUUtil = v
		}
		i += 2
	}
	// remaining tokens: temperatures (key@valueC) and power (key value/total)
	for i < len(fields) {
		tok := fields[i]
		// temperature entries
		if parts := strings.SplitN(tok, "@", 2); len(parts) == 2 && strings.HasSuffix(parts[1], "C") {
			key := parts[0]
			if v, _, err := parseValueUnit(parts[1]); err == nil {
				s.TemperaturesC[key] = v
			}
		} else if i+1 < len(fields) && strings.Contains(fields[i+1], "/") {
			// power entries
			key := tok
			usedTotal := fields[i+1]
			// The value/total format might not apply universally, check specifically for mW
			if strings.HasSuffix(usedTotal, "mW") {
				if kt := strings.SplitN(usedTotal, "/", 2); len(kt) == 2 {
					if v, _, err := parseValueUnit(kt[0]); err == nil {
						s.PowerMW[key] = v
					}
					// We only parse the first value (current consumption) for now
				}
			}
			i++ // Consume the power value field
		}
		i++ // Consume the key field or temperature field
	}

	return s, nil
}
