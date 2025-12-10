package wrapper

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewCobrLambdaHandler_BasicExecution(t *testing.T) {
	// Create a test command
	var name string
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Hello, %s!\n", name)
			fmt.Println("Command executed successfully")
		},
	}
	cmd.Flags().StringVar(&name, "name", "World", "Name to greet")

	// Create the handler
	handler := NewCobrLambdaHandler(cmd)

	// Create a test event
	event := CobraLambdaEvent{
		Args: []string{"--name", "Lambda"},
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	// Execute the handler
	ctx := context.Background()
	result, err := handler(ctx, eventJSON)

	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Check the result is an OutputCapture
	output, ok := result.(*CobraLambdaOutput)
	if !ok {
		t.Fatalf("Expected *OutputCapture, got %T", result)
	}

	// Verify output
	if !strings.Contains(output.Stdout, "Hello, Lambda!") {
		t.Errorf("Expected output to contain 'Hello, Lambda!', got: %s", output.Stdout)
	}
	if !strings.Contains(output.Stdout, "Command executed successfully") {
		t.Errorf("Expected output to contain 'Command executed successfully', got: %s", output.Stdout)
	}
}

func TestNewCobrLambdaHandler_InvalidJSON(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("This should not run")
		},
	}

	handler := NewCobrLambdaHandler(cmd)

	// Pass invalid JSON
	invalidJSON := json.RawMessage(`{"args": [invalid json}`)

	ctx := context.Background()
	result, err := handler(ctx, invalidJSON)

	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}

	if result != nil {
		t.Errorf("Expected nil result on error, got: %v", result)
	}
}

func TestNewCobrLambdaHandler_CommandError(t *testing.T) {
	// Create a command that returns an error
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("About to fail")
			return fmt.Errorf("command execution failed")
		},
	}

	handler := NewCobrLambdaHandler(cmd)

	event := CobraLambdaEvent{
		Args: []string{},
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	ctx := context.Background()
	result, err := handler(ctx, eventJSON)

	if err == nil {
		t.Fatal("Expected error from command, got nil")
	}

	if err.Error() != "command execution failed" {
		t.Errorf("Expected 'command execution failed', got: %v", err)
	}

	// Result should still contain output capture
	output, ok := result.(*CobraLambdaOutput)
	if !ok {
		t.Fatalf("Expected *OutputCapture even on error, got %T", result)
	}

	// Verify output was captured before error
	if !strings.Contains(output.Stdout, "About to fail") {
		t.Errorf("Expected output to be captured before error, got: %s", output.Stdout)
	}
}

func TestNewCobrLambdaHandler_WithSubcommands(t *testing.T) {
	// Create a root command with subcommands
	rootCmd := &cobra.Command{
		Use: "root",
	}

	var value string
	subCmd := &cobra.Command{
		Use: "process",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("Processing: %s\n", value)
			for i, arg := range args {
				cmd.Printf("Arg %d: %s\n", i, arg)
			}
		},
	}
	subCmd.Flags().StringVar(&value, "value", "", "Value to process")

	rootCmd.AddCommand(subCmd)

	handler := NewCobrLambdaHandler(rootCmd)

	event := CobraLambdaEvent{
		Args: []string{"process", "--value", "test-data", "extra", "args"},
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	ctx := context.Background()
	result, err := handler(ctx, eventJSON)

	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	output, ok := result.(*CobraLambdaOutput)
	if !ok {
		t.Fatalf("Expected *OutputCapture, got %T", result)
	}

	if !strings.Contains(output.Stdout, "Processing: test-data") {
		t.Errorf("Expected subcommand output, got: %s", output.Stdout)
	}
	if !strings.Contains(output.Stdout, "Arg 0: extra") {
		t.Errorf("Expected args to be passed, got: %s", output.Stdout)
	}
	if !strings.Contains(output.Stdout, "Arg 1: args") {
		t.Errorf("Expected args to be passed, got: %s", output.Stdout)
	}
}

func TestNewCobrLambdaHandler_EmptyArgs(t *testing.T) {
	executed := false
	cmd := &cobra.Command{
		Use: "test",
		Run: func(cmd *cobra.Command, args []string) {
			executed = true
			cmd.Println("Executed with no args")
		},
	}

	handler := NewCobrLambdaHandler(cmd)

	event := CobraLambdaEvent{
		Args: []string{},
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	ctx := context.Background()
	result, err := handler(ctx, eventJSON)

	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if !executed {
		t.Error("Command was not executed")
	}

	output, ok := result.(*CobraLambdaOutput)
	if !ok {
		t.Fatalf("Expected *OutputCapture, got %T", result)
	}

	if !strings.Contains(output.Stdout, "Executed with no args") {
		t.Errorf("Expected output, got: %s", output.Stdout)
	}
}

func TestNewCobrLambdaHandler_ContextPropagation(t *testing.T) {
	type contextKey string
	const testKey contextKey = "test-key"

	// Create a command that checks context
	receivedCtx := false
	cmd := &cobra.Command{
		Use: "test",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			if ctx != nil {
				if val := ctx.Value(testKey); val == "test-value" {
					receivedCtx = true
					cmd.Println("Context received correctly")
				}
			}
		},
	}

	handler := NewCobrLambdaHandler(cmd)

	event := CobraLambdaEvent{
		Args: []string{},
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	// Create context with value
	ctx := context.WithValue(context.Background(), testKey, "test-value")
	result, err := handler(ctx, eventJSON)

	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if !receivedCtx {
		t.Error("Context was not properly propagated to command")
	}

	output, ok := result.(*CobraLambdaOutput)
	if !ok {
		t.Fatalf("Expected *OutputCapture, got %T", result)
	}

	if !strings.Contains(output.Stdout, "Context received correctly") {
		t.Errorf("Expected context propagation confirmation, got: %s", output.Stdout)
	}
}
