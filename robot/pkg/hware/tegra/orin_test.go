package tegra

import "testing"

func TestOrinParser(t *testing.T) {
	parser := OrinParser{}
	stats, err := parser.Parse(OrinExample)
	if err != nil {
		t.Fatalf("failed to parse OrinExample: %v", err)
	}
	if stats.Timestamp.Format("01-02-2006 15:04:05") != "05-04-2025 12:59:49" {
		t.Errorf("expected timestamp 05-04-2025 12:59:49, got %s", stats.Timestamp)
	}
	if stats.RAM_MB.Used != 3492 || stats.RAM_MB.Total != 62840 {
		t.Errorf("expected RAM 3492/62840MB, got %v", stats.RAM_MB)
	}
	if stats.Swap_MB.Used != 0 || stats.Swap_MB.Total != 31420 {
		t.Errorf("expected SWAP 0/31420MB, got %v", stats.Swap_MB)
	}
	if len(stats.CPUUtil) != 12 {
		t.Errorf("expected 12 CPU utilization values, got %d", len(stats.CPUUtil))
	}
	if stats.TemperaturesC[OrinTempCPU] != 49.968 {
		t.Errorf("expected CPU temperature 49.968C, got %v", stats.TemperaturesC[OrinTempCPU])
	}
	if stats.TemperaturesC[OrinTempSOC0] != 47.562 {
		t.Errorf("expected SOC0 temperature 47.562, got %v", stats.TemperaturesC[OrinTempSOC0])
	}
	if stats.TemperaturesC[OrinTempSOC1] != 46.812 {
		t.Errorf("expected SOC1 temperature 46.812, got %v", stats.TemperaturesC[OrinTempSOC1])
	}
	if stats.TemperaturesC[OrinTempSOC2] != 47 {
		t.Errorf("expected SOC2 temperature 47C, got %v", stats.TemperaturesC[OrinTempSOC2])
	}
}
