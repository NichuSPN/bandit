package config

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type HardwareMemoryInfo struct {
	TotalRAMBytes uint64
	TotalRAMGB    float64
	OSName        string
	Arch          string
}

type VRAMRecommendation struct {
	SuggestedNumGPU    int
	SuggestedVRAMLimit string
	SuggestedModel     string
	Reason             string
}

func GetSystemMemoryInfo() HardwareMemoryInfo {
	info := HardwareMemoryInfo{
		OSName: runtime.GOOS,
		Arch:   runtime.GOARCH,
	}

	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err == nil {
			bytesVal, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
			if err == nil {
				info.TotalRAMBytes = bytesVal
				info.TotalRAMGB = float64(bytesVal) / (1024 * 1024 * 1024)
			}
		}
	case "linux":
		out, err := os.ReadFile("/proc/meminfo")
		if err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "MemTotal:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						kbVal, err := strconv.ParseUint(fields[1], 10, 64)
						if err == nil {
							info.TotalRAMBytes = kbVal * 1024
							info.TotalRAMGB = float64(info.TotalRAMBytes) / (1024 * 1024 * 1024)
						}
					}
					break
				}
			}
		}
	}

	if info.TotalRAMGB == 0 {
		info.TotalRAMGB = 16.0 // Fallback estimate
	}

	return info
}

func SuggestVRAMSetting(info HardwareMemoryInfo, currentModel string) VRAMRecommendation {
	rec := VRAMRecommendation{
		SuggestedModel: currentModel,
	}

	ram := info.TotalRAMGB

	if ram <= 8.5 {
		rec.SuggestedNumGPU = 12
		rec.SuggestedVRAMLimit = "4GB"
		rec.SuggestedModel = "qwen2.5-coder:7b"
		rec.Reason = fmt.Sprintf("System memory is %.1f GB. For <= 8GB RAM machines, setting VRAM to 4GB (12 GPU layers) and using a 7B model prevents OOM crashes.", ram)
	} else if ram <= 17.0 {
		rec.SuggestedNumGPU = 24
		rec.SuggestedVRAMLimit = "8GB"
		rec.SuggestedModel = "qwen2.5-coder:7b"
		rec.Reason = fmt.Sprintf("System memory is %.1f GB. Setting VRAM to 8GB (24 GPU layers) ensures smooth multi-turn performance.", ram)
	} else {
		rec.SuggestedNumGPU = -1
		rec.SuggestedVRAMLimit = "off"
		rec.Reason = fmt.Sprintf("System memory is %.1f GB. You have ample memory for full GPU offloading with no restrictions.", ram)
	}

	return rec
}
