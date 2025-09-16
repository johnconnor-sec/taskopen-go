package output

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// Utility functions
func isColorSupported() bool {
	// Check for explicit accessibility settings first
	if accessibility := os.Getenv("TASKOPEN_ACCESSIBILITY"); accessibility != "" {
		switch accessibility {
		case "screen-reader", "minimal":
			return false
		case "high-contrast":
			return true
		}
	}

	// Respect NO_COLOR standard
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Force color if requested
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}

	// Check if running in CI/CD environments
	if isCIEnvironment() {
		// Most CI environments support color
		return os.Getenv("CI_NO_COLOR") == ""
	}

	// Check for common accessibility tools
	if os.Getenv("NVDA") != "" || os.Getenv("JAWS") != "" || os.Getenv("ORCA") != "" {
		return false
	}

	// Check terminal type
	term := os.Getenv("TERM")
	if term == "" || term == "dumb" {
		return false
	}

	// Enhanced terminal detection
	colorTerms := []string{
		"xterm", "xterm-256color", "screen", "screen-256color",
		"tmux", "tmux-256color", "rxvt", "rxvt-unicode",
		"linux", "cygwin", "putty",
	}

	for _, colorTerm := range colorTerms {
		if strings.Contains(term, colorTerm) {
			return true
		}
	}

	// Check if stderr is a terminal
	return isTerminal(os.Stderr)
}

// isCIEnvironment checks if running in CI/CD
func isCIEnvironment() bool {
	ciVars := []string{
		"CI", "GITHUB_ACTIONS", "TRAVIS", "CIRCLECI", "GITLAB_CI",
		"JENKINS_URL", "BUILDKITE", "APPVEYOR", "DRONE", "TF_BUILD",
	}

	for _, env := range ciVars {
		if os.Getenv(env) != "" {
			return true
		}
	}
	return false
}

// isTerminal checks if a file descriptor is a terminal
func isTerminal(f *os.File) bool {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		f.Fd(),
		uintptr(syscall.TCGETS),
		0)
	return errno == 0
}

// Dynamic terminal width detection with fallback
func getTerminalWidth() int {
	// First check environment variable
	if width := os.Getenv("COLUMNS"); width != "" {
		if w, err := strconv.Atoi(width); err == nil && w > 0 {
			return w
		}
	}

	// Try to get terminal width from system call
	if width := getTerminalWidthSyscall(); width > 0 {
		return width
	}

	// Fallback to reasonable default
	return 80
}

// getTerminalWidthSyscall gets terminal width using system calls
func getTerminalWidthSyscall() int {
	type winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}

	ws := &winsize{}
	retCode, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stderr),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 {
		return 0
	}
	return int(ws.Col)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
