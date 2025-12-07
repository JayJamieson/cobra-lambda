package wrapper

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync"

	"github.com/spf13/cobra"
)

// OutputCapture holds captured output from both Cobra command and os.Stdout/Stderr
type OutputCapture struct {
	Output string
}

type threadSafeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (t *threadSafeBuffer) Write(p []byte) (n int, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.buf.Write(p)
}

func (t *threadSafeBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.buf.String()
}

type CobraLambda struct {
	cmd            *cobra.Command
	originalStdout *os.File
	originalStderr *os.File
	ctx            context.Context
	mu             sync.Mutex
}

func NewCobraLambda(ctx context.Context, cmd *cobra.Command) *CobraLambda {
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
		io.Copy(sharedBuffer, stdoutReader)
		done <- true
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(sharedBuffer, stderrReader)
		done <- true
	}()

	w.cmd.SetOut(sharedBuffer)
	w.cmd.SetErr(sharedBuffer)
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
		Output: sharedBuffer.String(),
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
