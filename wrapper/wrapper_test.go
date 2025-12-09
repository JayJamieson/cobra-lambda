package wrapper

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
)

func TestCobraWrapper_Execute(t *testing.T) {
	// Create a test command
	var name string
	var age int
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
		Run: func(cmd *cobra.Command, args []string) {
			// Write to cobra output
			cmd.Println("Hello from Cobra!")
			cmd.Printf("Name: %s, Age: %d\n", name, age)

			// Write to os.Stdout
			fmt.Println("Hello from os.Stdout!")
			fmt.Fprintf(os.Stdout, "Additional stdout output\n")

			// Write to os.Stderr
			fmt.Fprintln(os.Stderr, "Error message to stderr")
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Name")
	cmd.Flags().IntVar(&age, "age", 0, "Age")

	// Wrap and execute
	wrapper := NewCobraLambdaCLI(context.TODO(), cmd)
	output, err := wrapper.Execute([]string{"--name", "Alice", "--age", "30"})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check all output in the shared buffer
	if !strings.Contains(output.Stdout, "Hello from Cobra!") {
		t.Errorf("Stdout missing expected text. Got: %s", output.Stdout)
	}
	if !strings.Contains(output.Stdout, "Name: Alice, Age: 30") {
		t.Errorf("Stdout missing expected text. Got: %s", output.Stdout)
	}
	if !strings.Contains(output.Stdout, "Hello from os.Stdout!") {
		t.Errorf("Stdout missing expected text. Got: %s", output.Stdout)
	}
	if !strings.Contains(output.Stdout, "Additional stdout output") {
		t.Errorf("Stdout missing expected text. Got: %s", output.Stdout)
	}
	if !strings.Contains(output.Stdout, "Error message to stderr") {
		t.Errorf("Stdout missing stderr text. Got: %s", output.Stdout)
	}
}

func TestCobraWrapper_ExecuteWithError(t *testing.T) {
	// Create a command that returns an error
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("Before error")
			fmt.Println("Stdout before error")
			return fmt.Errorf("command failed")
		},
	}

	wrapper := NewCobraLambdaCLI(context.TODO(), cmd)
	output, err := wrapper.Execute([]string{})

	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if err.Error() != "command failed" {
		t.Errorf("Expected 'command failed', got: %v", err)
	}

	// Stdout should still be captured even with error
	if !strings.Contains(output.Stdout, "Before error") {
		t.Errorf("Stdout not captured on error. Got: %s", output.Stdout)
	}
	if !strings.Contains(output.Stdout, "Stdout before error") {
		t.Errorf("Stdout not captured on error. Got: %s", output.Stdout)
	}
}

func TestCobraWrapper_StdoutStderrRestored(t *testing.T) {
	originalStdout := os.Stdout
	originalStderr := os.Stderr

	cmd := &cobra.Command{
		Use: "test",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Test output")
		},
	}

	wrapper := NewCobraLambdaCLI(context.TODO(), cmd)
	_, err := wrapper.Execute([]string{})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify stdout and stderr are restored
	if os.Stdout != originalStdout {
		t.Error("os.Stdout was not restored")
	}
	if os.Stderr != originalStderr {
		t.Error("os.Stderr was not restored")
	}
}

func TestCobraWrapper_ThreadSafety(t *testing.T) {
	// Create a command that writes a lot
	cmd := &cobra.Command{
		Use: "test",
		Run: func(cmd *cobra.Command, args []string) {
			for i := 0; i < 100; i++ {
				cmd.Printf("Cobra line %d\n", i)
				fmt.Printf("Stdout line %d\n", i)
			}
		},
	}

	wrapper := NewCobraLambdaCLI(context.TODO(), cmd)

	// Run multiple times concurrently
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			output, err := wrapper.Execute([]string{})
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: %v", id, err)
				return
			}
			// Verify output has content
			if len(output.Stdout) == 0 {
				errors <- fmt.Errorf("goroutine %d: empty Stdout", id)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}
}

func TestCobraWrapper_SubcommandExecution(t *testing.T) {
	// Create a root command with subcommands
	rootCmd := &cobra.Command{
		Use: "root",
	}

	subCmd := &cobra.Command{
		Use: "sub",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("Subcommand executed")
			fmt.Println("Subcommand stdout")
		},
	}

	rootCmd.AddCommand(subCmd)

	wrapper := NewCobraLambdaCLI(context.TODO(), rootCmd)
	output, err := wrapper.Execute([]string{"sub"})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(output.Stdout, "Subcommand executed") {
		t.Errorf("Subcommand output not captured. Got: %s", output.Stdout)
	}
	if !strings.Contains(output.Stdout, "Subcommand stdout") {
		t.Errorf("Subcommand stdout not captured. Got: %s", output.Stdout)
	}
}

func TestCobraWrapper_EmptyStdout(t *testing.T) {
	// Command that produces no output
	cmd := &cobra.Command{
		Use: "test",
		Run: func(cmd *cobra.Command, args []string) {
			// Do nothing
		},
	}

	wrapper := NewCobraLambdaCLI(context.TODO(), cmd)
	output, err := wrapper.Execute([]string{})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if output.Stdout != "" {
		t.Errorf("Expected empty Stdout, got: %s", output.Stdout)
	}
}
