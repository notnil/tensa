package tegra

import (
	"fmt"
	"strings"
	"time"
)

const OrinExample = "05-04-2025 12:59:49 RAM 3492/62840MB (lfb 35x4MB) SWAP 0/31420MB (cached 0MB) CPU [30%@729,3%@729,4%@729,4%@729,4%@1267,7%@1267,4%@1267,3%@1267,3%@1497,4%@1497,4%@1497,5%@1497] GR3D_FREQ 0% cpu@49.968C soc2@47C soc0@47.562C tj@49.812C soc1@46.812C VDD_GPU_SOC 3124mW/3124mW VDD_CPU_CV 720mW/720mW VIN_SYS_5V0 5140mW/5140mW"

const (
	OrinTempCPU  = "cpu"
	OrinTempSOC0 = "soc0"
	OrinTempSOC1 = "soc1"
	OrinTempSOC2 = "soc2"
	OrinTempTJ   = "tj"
)

// OrinParser implements Parser for Orin tegrastats output.
type OrinParser struct{}

// Parse takes a single tegrastats line from an Orin device and returns a populated Stats.
func (p OrinParser) Parse(line string) (Stats, error) {
	var s Stats
	// s.OtherMetrics = make(map[string]string)
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
				if v, _, err := parseValueUnit(parts[0]); err == nil {
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
			if kt := strings.SplitN(usedTotal, "/", 2); len(kt) == 2 {
				if v, _, err := parseValueUnit(kt[0]); err == nil {
					s.PowerMW[key] = v
				}
			}
			i++
		}
		i++
	}

	return s, nil
}

// parseValueUnit parses a string like "12.34C" or "56%" into its numeric value and unit.
func parseValueUnit(s string) (float64, string, error) {
	var v float64
	var unit string
	_, err := fmt.Sscanf(s, "%f%s", &v, &unit)
	return v, unit, err
}

// parseUsage parses a string like "123/456MB" into a UsageValueUnit.
func parseUsage(s string) (UsageValueUnit, error) {
	var used, total float64
	var unit string
	_, err := fmt.Sscanf(s, "%f/%f%s", &used, &total, &unit)
	if err != nil {
		return UsageValueUnit{}, err
	}
	return UsageValueUnit{Used: used, Total: total}, nil
}
