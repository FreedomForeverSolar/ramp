package cmd

import (
	"fmt"
	"os/exec"
)

// setPortEnvVars adds port environment variables to a command
// Sets RAMP_PORT (first port) and RAMP_PORT_1, RAMP_PORT_2, etc.
func setPortEnvVars(cmd *exec.Cmd, ports []int) {
	if len(ports) == 0 {
		return
	}

	// Set RAMP_PORT to first port (backward compatibility)
	cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_PORT=%d", ports[0]))

	// Set indexed ports (RAMP_PORT_1, RAMP_PORT_2, etc.)
	for i, port := range ports {
		cmd.Env = append(cmd.Env, fmt.Sprintf("RAMP_PORT_%d=%d", i+1, port))
	}
}
