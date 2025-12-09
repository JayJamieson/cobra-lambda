package wrapper

import (
	"context"
	"io"
	"os"
	"sync"

	"github.com/spf13/cobra"
)

// OutputCapture holds captured output from both Cobra command and os.Stdout/Stderr
type OutputCapture struct {
	Stdout string `json:"stdout"`
}

type CobraLambda struct {
	cmd            *cobra.Command
	originalStdout *os.File
	originalStderr *os.File
	ctx            context.Context
	mu             sync.Mutex
}

func NewCobraLambdaCLI(ctx context.Context, cmd *cobra.Command) *CobraLambda {
	cmd.SetContext(ctx)
	return &CobraLambda{
		cmd:            cmd,
		ctx:            ctx,
		originalStdout: os.Stdout,
		originalStderr: os.Stderr,
	}
}

// Execute runs the Cobra command with the given arguments and captures all output
// This method is thread-safe and will restore os.Stdout/Stderr even if the command panics
// Note: Only one execution can run at a time per wrapper instance to avoid interference
func (w *CobraLambda) Execute(args []string) (*OutputCapture, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	sharedBuffer := &threadSafeBuffer{}

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		stdoutWriter.Close()
		stdoutReader.Close()
		return nil, err
	}

	// Ensure we restore original stdout/stderr even on panic
	defer func() {
		os.Stdout = w.originalStdout
		os.Stderr = w.originalStderr
	}()

	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	done := make(chan bool, 2)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		mw := io.MultiWriter(sharedBuffer, w.originalStdout)
		io.Copy(mw, stdoutReader)
		done <- true
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		mw := io.MultiWriter(sharedBuffer, w.originalStderr)
		io.Copy(mw, stderrReader)
		done <- true
	}()

	// when set to nil, cobra will use stdout/stderr
	w.cmd.SetOut(nil)
	w.cmd.SetErr(nil)
	w.cmd.SetArgs(args)

	execErr := w.cmd.Execute()

	stdoutWriter.Close()
	stderrWriter.Close()

	wg.Wait()
	close(done)

	stdoutReader.Close()
	stderrReader.Close()

	os.Stdout = w.originalStdout
	os.Stderr = w.originalStderr

	return &OutputCapture{
		Stdout: sharedBuffer.String(),
	}, execErr
}

// ExecuteWithContext is a convenience method that runs Execute with the provided context overriding
// context passed in from NewCobraLambda and restoring to original context after execution
func (w *CobraLambda) ExecuteContext(ctx context.Context, args []string) (*OutputCapture, error) {
	w.cmd.SetContext(ctx)
	output, err := w.Execute(args)
	w.cmd.SetContext(w.ctx)
	return output, err
}
