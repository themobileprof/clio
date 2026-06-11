package platform

import (
	"os"
	"strconv"
	"strings"
)

const lowMemoryThresholdKB = 3 * 1024 * 1024 // 3 GB

// IsTermux reports whether the process is running inside Termux.
func IsTermux() bool {
	return os.Getenv("TERMUX_VERSION") != ""
}

// TotalMemoryKB returns total system RAM from /proc/meminfo, or 0 if unknown.
func TotalMemoryKB() int64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseInt(fields[1], 10, 64)
				if err == nil {
					return kb
				}
			}
			break
		}
	}
	return 0
}

// IsLowMemoryDevice reports whether the host likely has constrained RAM.
func IsLowMemoryDevice() bool {
	if IsTermux() {
		return true
	}
	total := TotalMemoryKB()
	return total > 0 && total < lowMemoryThresholdKB
}
