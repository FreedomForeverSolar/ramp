package operations

// ProgressReporter abstracts progress reporting for both CLI and UI contexts.
// CLI implementations use spinners; UI implementations broadcast WebSocket messages.
type ProgressReporter interface {
	// Start begins a new operation phase with the given message.
	Start(message string)

	// Update changes the current message without ending the phase.
	Update(message string)

	// UpdateWithProgress updates message and sets percentage (0-100).
	// CLI implementations may ignore the percentage.
	UpdateWithProgress(message string, percentage int)

	// Stop halts progress indication without a status message.
	// Use this before streaming command output to avoid visual conflicts.
	Stop()

	// Success ends the current phase successfully.
	Success(message string)

	// Error reports an error.
	Error(message string)

	// Warning reports a warning without stopping.
	Warning(message string)

	// Info reports informational message (may be verbose-only for CLI).
	Info(message string)

	// Complete marks the entire operation as finished.
	Complete(message string)
}

// OutputStreamer handles streaming output from commands.
// This is separate from ProgressReporter because commands need
// line-by-line output streaming, not status updates.
type OutputStreamer interface {
	// WriteLine sends a line of stdout output.
	WriteLine(line string)

	// WriteErrorLine sends a line of stderr output.
	WriteErrorLine(line string)
}

// ConfirmationHandler handles user confirmations (e.g., "delete with uncommitted changes?")
type ConfirmationHandler interface {
	// Confirm asks user for yes/no confirmation. Returns true if confirmed.
	Confirm(message string) bool
}
