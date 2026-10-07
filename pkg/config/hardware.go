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

func ParseModelParamSize(modelName string) float64 {
	m := strings.ToLower(modelName)
	if strings.Contains(m, "70b") {
		return 70.0
	}
	if strings.Contains(m, "32b") || strings.Contains(m, "33b") {
		return 32.0
	}
	if strings.Contains(m, "14b") || strings.Contains(m, "15b") {
		return 14.0
	}
	if strings.Contains(m, "7b") || strings.Contains(m, "8b") {
		return 7.0
	}
	if strings.Contains(m, "3b") {
		return 3.0
	}
	if strings.Contains(m, "1.5b") || strings.Contains(m, "1b") {
		return 1.5
	}
	return 7.0
}

func SuggestVRAMSetting(info HardwareMemoryInfo, currentModel string, paramSize float64) VRAMRecommendation {
	rec := VRAMRecommendation{
		SuggestedModel: currentModel,
	}

	ram := info.TotalRAMGB
	if paramSize <= 0 {
		paramSize = ParseModelParamSize(currentModel)
	}

	if paramSize >= 70.0 {
		if ram < 48.0 {
			rec.SuggestedNumGPU = 16
			rec.SuggestedVRAMLimit = "16GB"
			rec.SuggestedModel = "qwen3:14b"
			rec.Reason = fmt.Sprintf("System memory is %.1f GB. A 70B model requires ~45GB memory. We recommend switching to '%s' or offloading only 16 layers to prevent OOM.", ram, rec.SuggestedModel)
		} else {
			rec.SuggestedNumGPU = -1
			rec.SuggestedVRAMLimit = "off"
			rec.Reason = fmt.Sprintf("System memory is %.1f GB. You have sufficient memory for full offloading of 70B model '%s'.", ram, currentModel)
		}
	} else if paramSize >= 32.0 {
		if ram < 24.0 {
			rec.SuggestedNumGPU = 16
			rec.SuggestedVRAMLimit = "12GB"
			rec.SuggestedModel = "qwen2.5-coder:7b"
			rec.Reason = fmt.Sprintf("System memory is %.1f GB. A 32B model requires ~22GB VRAM. We recommend switching to '%s' or limiting VRAM to 12GB (16 GPU layers).", ram, rec.SuggestedModel)
		} else {
			rec.SuggestedNumGPU = -1
			rec.SuggestedVRAMLimit = "off"
			rec.Reason = fmt.Sprintf("System memory is %.1f GB. You have sufficient memory for full GPU offloading of 32B model '%s'.", ram, currentModel)
		}
	} else if paramSize >= 14.0 {
		if ram <= 8.5 {
			rec.SuggestedNumGPU = 10
			rec.SuggestedVRAMLimit = "4GB"
			rec.SuggestedModel = "qwen2.5-coder:7b"
			rec.Reason = fmt.Sprintf("System memory is %.1f GB. Running 14B model '%s' will cause OOM crashes. Recommend switching to 'qwen2.5-coder:7b' or limiting VRAM to 4GB (10 GPU layers).", ram, currentModel)
		} else if ram <= 17.0 {
			rec.SuggestedNumGPU = 20
			rec.SuggestedVRAMLimit = "8GB"
			rec.SuggestedModel = currentModel
			rec.Reason = fmt.Sprintf("System memory is %.1f GB. For 14B model '%s', limiting VRAM to 8GB (20 GPU layers) balances GPU speed while preserving host RAM for OS and context.", ram, currentModel)
		} else {
			rec.SuggestedNumGPU = -1
			rec.SuggestedVRAMLimit = "off"
			rec.Reason = fmt.Sprintf("System memory is %.1f GB. You have ample memory to run 14B model '%s' with full GPU offloading.", ram, currentModel)
		}
	} else {
		if ram <= 8.5 {
			rec.SuggestedNumGPU = 16
			rec.SuggestedVRAMLimit = "4GB"
			rec.SuggestedModel = "qwen2.5-coder:7b"
			rec.Reason = fmt.Sprintf("System memory is %.1f GB. For 7B model '%s', setting VRAM limit to 4GB (16 GPU layers) prevents system memory paging.", ram, currentModel)
		} else {
			rec.SuggestedNumGPU = -1
			rec.SuggestedVRAMLimit = "off"
			rec.Reason = fmt.Sprintf("System memory is %.1f GB. You have ample memory for full GPU offloading of '%s' with no restrictions.", ram, currentModel)
		}
	}

	return rec
}
