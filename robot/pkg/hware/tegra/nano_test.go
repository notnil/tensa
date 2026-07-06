package tegra

import "testing"

func TestNanoParser(t *testing.T) {
	parser := NanoParser{}
	stats, err := parser.Parse(NanoExample)
	if err != nil {
		t.Fatalf("failed to parse NanoExample: %v", err)
	}
	if stats.Timestamp.Format("01-02-2006 15:04:05") != "05-05-2025 06:05:39" {
		t.Errorf("expected timestamp 05-05-2025 06:05:39, got %s", stats.Timestamp)
	}
	if stats.RAM_MB.Used != 2639 || stats.RAM_MB.Total != 7620 {
		t.Errorf("expected RAM 2639/7620MB, got %v", stats.RAM_MB)
	}
	if stats.Swap_MB.Used != 0 || stats.Swap_MB.Total != 3810 {
		t.Errorf("expected SWAP 0/3810MB, got %v", stats.Swap_MB)
	}
	if len(stats.CPUUtil) != 6 {
		t.Errorf("expected 6 CPU utilization values, got %d", len(stats.CPUUtil))
	}
	if stats.TemperaturesC[NanoTempCPU] != 44.156 {
		t.Errorf("expected CPU temperature 44.156C, got %v", stats.TemperaturesC[NanoTempCPU])
	}
	if stats.TemperaturesC[NanoTempGPU] != 45.968 {
		t.Errorf("expected GPU temperature 45.968C, got %v", stats.TemperaturesC[NanoTempGPU])
	}
	if stats.TemperaturesC[NanoTempSOC0] != 44.687 {
		t.Errorf("expected SOC0 temperature 44.687C, got %v", stats.TemperaturesC[NanoTempSOC0])
	}
	if stats.TemperaturesC[NanoTempSOC1] != 45.312 {
		t.Errorf("expected SOC1 temperature 45.312C, got %v", stats.TemperaturesC[NanoTempSOC1])
	}
	if stats.TemperaturesC[NanoTempSOC2] != 43.781 {
		t.Errorf("expected SOC2 temperature 43.781C, got %v", stats.TemperaturesC[NanoTempSOC2])
	}
	if stats.TemperaturesC[NanoTempTJ] != 45.968 {
		t.Errorf("expected TJ temperature 45.968C, got %v", stats.TemperaturesC[NanoTempTJ])
	}
	if stats.PowerMW["VDD_IN"] != 5263 {
		t.Errorf("expected VDD_IN power 5263mW, got %v", stats.PowerMW["VDD_IN"])
	}
	if stats.PowerMW["VDD_CPU_GPU_CV"] != 717 {
		t.Errorf("expected VDD_CPU_GPU_CV power 717mW, got %v", stats.PowerMW["VDD_CPU_GPU_CV"])
	}
	if stats.PowerMW["VDD_SOC"] != 1794 {
		t.Errorf("expected VDD_SOC power 1794mW, got %v", stats.PowerMW["VDD_SOC"])
	}
}
