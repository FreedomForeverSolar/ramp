package ui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/briandowns/spinner"
)

var (
	Verbose = false
)

type ProgressUI struct {
	spinner *spinner.Spinner
}

func NewProgress() *ProgressUI {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Color("cyan")
	return &ProgressUI{spinner: s}
}

func (p *ProgressUI) Start(message string) {
	if Verbose {
		fmt.Printf("%s\n", message)
		return
	}
	p.spinner.Suffix = " " + message
	p.spinner.Start()
}

func (p *ProgressUI) Success(message string) {
	if Verbose {
		fmt.Printf("✓ %s\n", message)
		return
	}
	p.spinner.Stop()
	fmt.Printf("✓ %s\n", message)
}

func (p *ProgressUI) Warning(message string) {
	if Verbose {
		fmt.Printf("Warning: %s\n", message)
		return
	}
	p.spinner.Stop()
	fmt.Printf("Warning: %s\n", message)
}

func (p *ProgressUI) Error(message string) {
	if Verbose {
		fmt.Printf("Error: %s\n", message)
		return
	}
	p.spinner.Stop()
	fmt.Printf("Error: %s\n", message)
}

func (p *ProgressUI) Info(message string) {
	if Verbose {
		fmt.Printf("  %s\n", message)
	}
	// In non-verbose mode, don't print info messages to avoid clutter
}

func (p *ProgressUI) Stop() {
	if !Verbose {
		p.spinner.Stop()
	}
}

// Update changes the spinner message without stopping it
func (p *ProgressUI) Update(message string) {
	if Verbose {
		fmt.Printf("%s\n", message)
		return
	}
	p.spinner.Suffix = " " + message
}

func WithProgress(message string, fn func() error) error {
	progress := NewProgress()
	progress.Start(message)
	
	err := fn()
	
	if err != nil {
		progress.Error(fmt.Sprintf("%s failed: %v", message, err))
	} else {
		progress.Success(message)
	}
	
	return err
}

type OutputCapture struct {
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func (oc *OutputCapture) GetStdout() io.Writer {
	return &oc.stdout
}

func (oc *OutputCapture) GetStderr() io.Writer {
	return &oc.stderr
}

func (oc *OutputCapture) HasOutput() bool {
	return oc.stdout.Len() > 0 || oc.stderr.Len() > 0
}

func (oc *OutputCapture) PrintOutput() {
	if oc.stdout.Len() > 0 {
		fmt.Print(oc.stdout.String())
	}
	if oc.stderr.Len() > 0 {
		fmt.Fprint(os.Stderr, oc.stderr.String())
	}
}

func RunCommandWithProgress(cmd *exec.Cmd, message string) error {
	if Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Printf("%s\n", message)
		return cmd.Run()
	}

	progress := NewProgress()
	progress.Start(message)

	capture := &OutputCapture{}
	cmd.Stdout = capture.GetStdout()
	cmd.Stderr = capture.GetStderr()

	err := cmd.Run()

	progress.Stop()

	if err != nil {
		progress.Error(fmt.Sprintf("%s failed", message))
		if capture.HasOutput() {
			fmt.Println("\nOutput:")
			capture.PrintOutput()
		}
	} else {
		// Print the output first (if any), then show success message
		if capture.HasOutput() {
			capture.PrintOutput()
		}
		progress.Success(message)
	}

	return err
}