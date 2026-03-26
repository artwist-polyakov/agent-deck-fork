package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Platform represents the detected platform
type Platform string

const (
	PlatformMacOS   Platform = "macos"
	PlatformLinux   Platform = "linux"
	PlatformWSL1    Platform = "wsl1"
	PlatformWSL2    Platform = "wsl2"
	PlatformWindows Platform = "windows"
	PlatformUnknown Platform = "unknown"
)

// cached detection result
var detectedPlatform Platform
var detectionDone bool

// Detect returns the current platform, caching the result
func Detect() Platform {
	if detectionDone {
		return detectedPlatform
	}

	detectedPlatform = detectPlatform()
	detectionDone = true
	return detectedPlatform
}

// detectPlatform performs the actual platform detection
func detectPlatform() Platform {
	switch runtime.GOOS {
	case "darwin":
		return PlatformMacOS
	case "windows":
		return PlatformWindows
	case "linux":
		// Could be native Linux or WSL - check further
		return detectLinuxOrWSL()
	default:
		return PlatformUnknown
	}
}

// detectLinuxOrWSL distinguishes between native Linux and WSL (1 or 2)
func detectLinuxOrWSL() Platform {
	// Quick check: WSL_DISTRO_NAME is set in WSL environments
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return detectWSLVersion()
	}

	// Fallback: Check /proc/version for WSL signatures
	procVersion, err := os.ReadFile("/proc/version")
	if err != nil {
		return PlatformLinux // Can't read, assume native Linux
	}

	versionStr := string(procVersion)

	// Check for WSL signatures
	if strings.Contains(versionStr, "microsoft") || strings.Contains(versionStr, "Microsoft") {
		return detectWSLVersion()
	}

	return PlatformLinux
}

// detectWSLVersion distinguishes between WSL1 and WSL2
func detectWSLVersion() Platform {
	// Method 1: Check /proc/version for WSL2 signature
	// WSL2 typically has "microsoft-standard-WSL2" or just lowercase "microsoft-standard"
	// WSL1 has "Microsoft" (capital M) without "standard"
	procVersion, err := os.ReadFile("/proc/version")
	if err == nil {
		versionStr := string(procVersion)

		// WSL2 signatures (lowercase "microsoft-standard")
		if strings.Contains(versionStr, "microsoft-standard") {
			return PlatformWSL2
		}

		// WSL1 signature (uppercase "Microsoft" without "standard")
		if strings.Contains(versionStr, "Microsoft") {
			return PlatformWSL1
		}
	}

	// Method 2: Check for WSL2-specific paths
	// /run/WSL exists only in WSL2
	if _, err := os.Stat("/run/WSL"); err == nil {
		return PlatformWSL2
	}

	// Method 3: Check for WSL interop
	// /proc/sys/fs/binfmt_misc/WSLInterop exists in both, but behavior differs
	// In WSL2, we also have /dev/vsock which is virtualization-specific
	if _, err := os.Stat("/dev/vsock"); err == nil {
		return PlatformWSL2
	}

	// Default to WSL1 if we detected WSL but can't determine version
	// (safer to assume WSL1 since it has more limitations)
	return PlatformWSL1
}

// IsWSL returns true if running in any WSL environment
func IsWSL() bool {
	p := Detect()
	return p == PlatformWSL1 || p == PlatformWSL2
}

// IsWSL1 returns true if running specifically in WSL1
func IsWSL1() bool {
	return Detect() == PlatformWSL1
}

// IsWSL2 returns true if running specifically in WSL2
func IsWSL2() bool {
	return Detect() == PlatformWSL2
}

// SupportsUnixSockets returns true if the platform reliably supports Unix domain sockets
func SupportsUnixSockets() bool {
	p := Detect()
	switch p {
	case PlatformMacOS, PlatformLinux, PlatformWSL2:
		return true
	case PlatformWSL1, PlatformWindows, PlatformUnknown:
		return false
	default:
		return false
	}
}

// String returns a human-readable platform name
func (p Platform) String() string {
	switch p {
	case PlatformMacOS:
		return "macOS"
	case PlatformLinux:
		return "Linux"
	case PlatformWSL1:
		return "WSL1"
	case PlatformWSL2:
		return "WSL2"
	case PlatformWindows:
		return "Windows"
	default:
		return "Unknown"
	}
}

// CheckFsnotifySupport checks if a path's filesystem supports fsnotify events reliably.
// Returns a warning message if on a problematic filesystem (9p, nfs, cifs, sshfs),
// or an empty string if fsnotify should work normally.
// This helps users understand why auto-reload might not work in WSL2 or network mounts.
func CheckFsnotifySupport(path string) string {
	// Only relevant on Linux (WSL2 uses 9p for Windows filesystem access)
	if runtime.GOOS != "linux" {
		return ""
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return ""
	}

	mounts, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return "" // Can't read mounts, assume OK
	}

	// Parse /proc/mounts to find filesystem type for the path
	// Format: device mountpoint fstype options ...
	// We need to find the longest matching mountpoint for our path
	var matchedMount, matchedFsType string
	for _, line := range strings.Split(string(mounts), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		mountPoint := fields[1]
		fsType := fields[2]

		// Check if this mount contains our path (use longest match)
		if strings.HasPrefix(absPath, mountPoint) {
			if len(mountPoint) > len(matchedMount) {
				matchedMount = mountPoint
				matchedFsType = fsType
			}
		}
	}

	if matchedFsType == "" {
		return ""
	}

	// Check for problematic filesystem types
	switch {
	case matchedFsType == "9p":
		return "Storage on 9p mount (WSL2 Windows filesystem): fsnotify disabled. Use Ctrl+R to refresh."
	case matchedFsType == "nfs" || matchedFsType == "nfs4":
		return "Storage on NFS mount: fsnotify may be unreliable. Use Ctrl+R to refresh."
	case matchedFsType == "cifs" || matchedFsType == "smbfs":
		return "Storage on CIFS/SMB mount: fsnotify may be unreliable. Use Ctrl+R to refresh."
	case strings.HasPrefix(matchedFsType, "fuse.sshfs"):
		return "Storage on SSHFS mount: fsnotify disabled. Use Ctrl+R to refresh."
	}

	return ""
}
