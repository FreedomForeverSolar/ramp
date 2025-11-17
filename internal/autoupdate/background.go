package autoupdate

import (
	"fmt"
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

	// Ensure parent directory exists
	dir, _ := getRampDir()
	os.MkdirAll(dir, 0755)

	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return // Can't open log file, skip
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start and immediately forget (don't wait)
	if err := cmd.Start(); err != nil {
		// Log the error before closing
		logFile.WriteString(fmt.Sprintf("Failed to spawn background checker: %v\n", err))
		logFile.Close()
		return
	}

	// Close the file in parent process - child has already inherited the file descriptor
	logFile.Close()
	// Note: Intentionally not calling cmd.Wait() - this is fire-and-forget
}
