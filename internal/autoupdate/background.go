package autoupdate

import (
	"os"
	"os/exec"
	"syscall"
)

// SpawnBackgroundChecker spawns a detached background process to check for updates.
// This function returns immediately and does not block the caller.
func SpawnBackgroundChecker() {
	// Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		return // Silently fail
	}

	// Re-exec current binary with internal flag
	cmd := exec.Command(exePath, "__internal_update_check")

	// Detach completely from parent process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // New process group (survives parent exit)
	}

	// Redirect output to log file
	logPath := getLogPath()
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return // Can't open log file, skip
	}
	// Note: Don't close logFile here - background process needs it

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start and immediately forget (don't wait)
	cmd.Start()
	// Note: Intentionally not calling cmd.Wait() - this is fire-and-forget
}
