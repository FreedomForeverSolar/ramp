package ui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// captureOutput captures stdout and stderr during function execution
func captureOutput(f func()) (stdout, stderr string) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	outC := make(chan string)
	errC := make(chan string)

	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, rOut)
		outC <- buf.String()
	}()

	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, rErr)
		errC <- buf.String()
	}()

	f()

	wOut.Close()
	wErr.Close()

	stdout = <-outC
	stderr = <-errC

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return
}

func TestNewProgress(t *testing.T) {
	p := NewProgress()
	if p == nil {
		t.Fatal("NewProgress returned nil")
	}
	if p.spinner == nil {
		t.Error("spinner not initialized")
	}
}

func TestProgressStart(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		message string
	}{
		{"verbose mode", true, "starting task"},
		{"non-verbose mode", false, "starting task"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVerbose := Verbose
			Verbose = tt.verbose
			defer func() { Verbose = oldVerbose }()

			p := NewProgress()
			stdout, _ := captureOutput(func() {
				p.Start(tt.message)
				p.Stop() // Stop immediately to avoid hanging
			})

			if tt.verbose && !strings.Contains(stdout, tt.message) {
				t.Errorf("Expected message %q in verbose output, got: %s", tt.message, stdout)
			}
		})
	}
}

func TestProgressSuccess(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		message string
	}{
		{"verbose mode", true, "task completed"},
		{"non-verbose mode", false, "task completed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVerbose := Verbose
			Verbose = tt.verbose
			defer func() { Verbose = oldVerbose }()

			p := NewProgress()
			stdout, _ := captureOutput(func() {
				p.Success(tt.message)
			})

			if !strings.Contains(stdout, "✓") {
				t.Error("Expected checkmark in success output")
			}
			if !strings.Contains(stdout, tt.message) {
				t.Errorf("Expected message %q in output, got: %s", tt.message, stdout)
			}
		})
	}
}

func TestProgressWarning(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		message string
	}{
		{"verbose mode", true, "warning message"},
		{"non-verbose mode", false, "warning message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVerbose := Verbose
			Verbose = tt.verbose
			defer func() { Verbose = oldVerbose }()

			p := NewProgress()
			stdout, _ := captureOutput(func() {
				p.Warning(tt.message)
			})

			if !strings.Contains(stdout, "Warning:") {
				t.Error("Expected 'Warning:' prefix in output")
			}
			if !strings.Contains(stdout, tt.message) {
				t.Errorf("Expected message %q in output, got: %s", tt.message, stdout)
			}
		})
	}
}

func TestProgressError(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		message string
	}{
		{"verbose mode", true, "error message"},
		{"non-verbose mode", false, "error message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVerbose := Verbose
			Verbose = tt.verbose
			defer func() { Verbose = oldVerbose }()

			p := NewProgress()
			stdout, _ := captureOutput(func() {
				p.Error(tt.message)
			})

			if !strings.Contains(stdout, "Error:") {
				t.Error("Expected 'Error:' prefix in output")
			}
			if !strings.Contains(stdout, tt.message) {
				t.Errorf("Expected message %q in output, got: %s", tt.message, stdout)
			}
		})
	}
}

func TestProgressInfo(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		message string
		expect  bool // expect output
	}{
		{"verbose mode", true, "info message", true},
		{"non-verbose mode", false, "info message", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVerbose := Verbose
			Verbose = tt.verbose
			defer func() { Verbose = oldVerbose }()

			p := NewProgress()
			stdout, _ := captureOutput(func() {
				p.Info(tt.message)
			})

			hasOutput := strings.Contains(stdout, tt.message)
			if tt.expect && !hasOutput {
				t.Errorf("Expected message %q in verbose output, got: %s", tt.message, stdout)
			}
			if !tt.expect && hasOutput {
				t.Errorf("Did not expect output in non-verbose mode, got: %s", stdout)
			}
		})
	}
}

func TestProgressUpdate(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		message string
	}{
		{"verbose mode", true, "updating task"},
		{"non-verbose mode", false, "updating task"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVerbose := Verbose
			Verbose = tt.verbose
			defer func() { Verbose = oldVerbose }()

			p := NewProgress()
			stdout, _ := captureOutput(func() {
				p.Start("initial message")
				p.Update(tt.message)
				p.Stop()
			})

			if tt.verbose && !strings.Contains(stdout, tt.message) {
				t.Errorf("Expected message %q in verbose output, got: %s", tt.message, stdout)
			}
		})
	}
}

func TestWithProgress(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		oldVerbose := Verbose
		Verbose = false
		defer func() { Verbose = oldVerbose }()

		called := false
		err := WithProgress("test operation", func() error {
			called = true
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if !called {
			t.Error("Expected function to be called")
		}
	})

	t.Run("failure", func(t *testing.T) {
		oldVerbose := Verbose
		Verbose = false
		defer func() { Verbose = oldVerbose }()

		expectedErr := fmt.Errorf("test error")
		err := WithProgress("test operation", func() error {
			return expectedErr
		})

		if err != expectedErr {
			t.Errorf("Expected error %v, got: %v", expectedErr, err)
		}
	})
}

func TestOutputCaptureGetters(t *testing.T) {
	oc := &OutputCapture{}

	if oc.GetStdout() == nil {
		t.Error("GetStdout returned nil")
	}
	if oc.GetStderr() == nil {
		t.Error("GetStderr returned nil")
	}
}

func TestOutputCaptureHasOutput(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		oc := &OutputCapture{}
		if oc.HasOutput() {
			t.Error("Expected no output for empty capture")
		}
	})

	t.Run("with stdout", func(t *testing.T) {
		oc := &OutputCapture{}
		fmt.Fprint(oc.GetStdout(), "test output")
		if !oc.HasOutput() {
			t.Error("Expected output after writing to stdout")
		}
	})

	t.Run("with stderr", func(t *testing.T) {
		oc := &OutputCapture{}
		fmt.Fprint(oc.GetStderr(), "test error")
		if !oc.HasOutput() {
			t.Error("Expected output after writing to stderr")
		}
	})

	t.Run("with both", func(t *testing.T) {
		oc := &OutputCapture{}
		fmt.Fprint(oc.GetStdout(), "test output")
		fmt.Fprint(oc.GetStderr(), "test error")
		if !oc.HasOutput() {
			t.Error("Expected output after writing to both")
		}
	})
}

func TestOutputCapturePrintOutput(t *testing.T) {
	t.Run("stdout only", func(t *testing.T) {
		oc := &OutputCapture{}
		fmt.Fprint(oc.GetStdout(), "test stdout")

		stdout, stderr := captureOutput(func() {
			oc.PrintOutput()
		})

		if !strings.Contains(stdout, "test stdout") {
			t.Errorf("Expected stdout to contain 'test stdout', got: %s", stdout)
		}
		if stderr != "" {
			t.Errorf("Expected empty stderr, got: %s", stderr)
		}
	})

	t.Run("stderr only", func(t *testing.T) {
		oc := &OutputCapture{}
		fmt.Fprint(oc.GetStderr(), "test stderr")

		stdout, stderr := captureOutput(func() {
			oc.PrintOutput()
		})

		if stdout != "" {
			t.Errorf("Expected empty stdout, got: %s", stdout)
		}
		if !strings.Contains(stderr, "test stderr") {
			t.Errorf("Expected stderr to contain 'test stderr', got: %s", stderr)
		}
	})

	t.Run("both stdout and stderr", func(t *testing.T) {
		oc := &OutputCapture{}
		fmt.Fprint(oc.GetStdout(), "test stdout")
		fmt.Fprint(oc.GetStderr(), "test stderr")

		stdout, stderr := captureOutput(func() {
			oc.PrintOutput()
		})

		if !strings.Contains(stdout, "test stdout") {
			t.Errorf("Expected stdout to contain 'test stdout', got: %s", stdout)
		}
		if !strings.Contains(stderr, "test stderr") {
			t.Errorf("Expected stderr to contain 'test stderr', got: %s", stderr)
		}
	})
}

func TestRunCommandWithProgress(t *testing.T) {
	t.Run("success in verbose mode", func(t *testing.T) {
		oldVerbose := Verbose
		Verbose = true
		defer func() { Verbose = oldVerbose }()

		cmd := exec.Command("echo", "test output")
		err := RunCommandWithProgress(cmd, "running test command")

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("success in non-verbose mode", func(t *testing.T) {
		oldVerbose := Verbose
		Verbose = false
		defer func() { Verbose = oldVerbose }()

		cmd := exec.Command("echo", "test output")
		stdout, _ := captureOutput(func() {
			err := RunCommandWithProgress(cmd, "running test command")
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})

		if !strings.Contains(stdout, "✓") {
			t.Error("Expected success checkmark in output")
		}
	})

	t.Run("failure in verbose mode", func(t *testing.T) {
		oldVerbose := Verbose
		Verbose = true
		defer func() { Verbose = oldVerbose }()

		cmd := exec.Command("sh", "-c", "exit 1")
		err := RunCommandWithProgress(cmd, "failing command")

		if err == nil {
			t.Error("Expected error for failing command")
		}
	})

	t.Run("failure in non-verbose mode", func(t *testing.T) {
		oldVerbose := Verbose
		Verbose = false
		defer func() { Verbose = oldVerbose }()

		cmd := exec.Command("sh", "-c", "exit 1")
		stdout, _ := captureOutput(func() {
			err := RunCommandWithProgress(cmd, "failing command")
			if err == nil {
				t.Error("Expected error for failing command")
			}
		})

		if !strings.Contains(stdout, "Error:") {
			t.Error("Expected error message in output")
		}
	})

	t.Run("captures command output on failure", func(t *testing.T) {
		oldVerbose := Verbose
		Verbose = false
		defer func() { Verbose = oldVerbose }()

		cmd := exec.Command("sh", "-c", "echo 'error details' >&2; exit 1")
		stdout, _ := captureOutput(func() {
			RunCommandWithProgress(cmd, "failing command")
		})

		if !strings.Contains(stdout, "Output:") {
			t.Error("Expected 'Output:' label when command fails with output")
		}
	})
}
