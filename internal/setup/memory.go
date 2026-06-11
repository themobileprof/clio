package setup

import "clio/internal/platform"

// TotalMemoryKB returns total system RAM from /proc/meminfo, or 0 if unknown.
func TotalMemoryKB() int64 {
	return platform.TotalMemoryKB()
}

// IsLowMemoryDevice reports whether the host likely has constrained RAM.
func IsLowMemoryDevice() bool {
	return platform.IsLowMemoryDevice()
}
