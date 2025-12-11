package operations

import "ramp/internal/ui"

// CLIProgressReporter implements ProgressReporter using the CLI spinner.
type CLIProgressReporter struct {
	progress *ui.ProgressUI
}

// NewCLIProgressReporter creates a progress reporter for CLI usage.
func NewCLIProgressReporter() *CLIProgressReporter {
	return &CLIProgressReporter{
		progress: ui.NewProgress(),
	}
}

func (r *CLIProgressReporter) Start(message string) {
	r.progress.Start(message)
}

func (r *CLIProgressReporter) Update(message string) {
	r.progress.Update(message)
}

func (r *CLIProgressReporter) UpdateWithProgress(message string, _ int) {
	// CLI ignores percentage - just update the message
	r.progress.Update(message)
}

func (r *CLIProgressReporter) Success(message string) {
	r.progress.Success(message)
}

func (r *CLIProgressReporter) Error(message string) {
	r.progress.Error(message)
}

func (r *CLIProgressReporter) Warning(message string) {
	r.progress.Warning(message)
}

func (r *CLIProgressReporter) Info(message string) {
	r.progress.Info(message)
}

func (r *CLIProgressReporter) Complete(message string) {
	r.progress.Success(message)
}

// CLIOutputStreamer implements OutputStreamer for CLI usage.
type CLIOutputStreamer struct{}

func (s *CLIOutputStreamer) WriteLine(line string) {
	println(line)
}

func (s *CLIOutputStreamer) WriteErrorLine(line string) {
	println(line)
}
