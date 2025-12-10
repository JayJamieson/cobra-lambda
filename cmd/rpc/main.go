package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/JayJamieson/cobra-lambda/wrapper"
	"github.com/aws/aws-lambda-go/lambda/messages"
)

const (
	lambdaServerPort = "8001"
	helpMessage      = `Usage: rpc [lambda-binary-path] [args...]

Runs a Go Lambda function locally over RPC.

Arguments:
  lambda-binary-path    Path to the compiled Lambda binary
  args...              Arguments to pass to the Lambda function

The Lambda binary will be started with _LAMBDA_SERVER_PORT=8001 and invoked over RPC.
`
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, helpMessage)
		os.Exit(1)
	}

	if os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Print(helpMessage)
		os.Exit(0)
	}

	lambdaPath := os.Args[1]
	lambdaArgs := []string{}
	if len(os.Args) > 2 {
		lambdaArgs = os.Args[2:]
	}

	// Check if lambda binary exists
	if _, err := os.Stat(lambdaPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Lambda binary not found at %s\n", lambdaPath)
		os.Exit(1)
	}

	// Start the Lambda process with _LAMBDA_SERVER_PORT environment variable
	cmd := exec.Command("go", "run", lambdaPath)
	cmd.Env = append(os.Environ(), fmt.Sprintf("_LAMBDA_SERVER_PORT=%s", lambdaServerPort))
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start Lambda process: %v\n", err)
		os.Exit(1)
	}

	// Ensure we kill the process on exit
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()

	// Wait for the Lambda server to be ready
	if err := waitForServer(lambdaServerPort, 5*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "Lambda server failed to start: %v\n", err)
		os.Exit(1)
	}

	// Connect to the Lambda RPC server
	client, err := rpc.Dial("tcp", fmt.Sprintf("localhost:%s", lambdaServerPort))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to Lambda server: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Prepare the invocation request
	argsEvent := wrapper.CobraLambdaEvent{Args: lambdaArgs}
	payload, err := json.Marshal(argsEvent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal event: %v\n", err)
		os.Exit(1)
	}

	args := messages.InvokeRequest{
		Payload: payload,
		Deadline: messages.InvokeRequest_Timestamp{
			Seconds: time.Now().Unix() + 10,
		},
	}

	// Invoke the Lambda function
	invokeResponse := &messages.InvokeResponse{}
	if err := client.Call("Function.Invoke", args, &invokeResponse); err != nil {
		fmt.Fprintf(os.Stderr, "Lambda invocation failed: %v\n", err)
		os.Exit(1)
	}

	// Check for Lambda execution errors
	if invokeResponse.Error != nil {
		fmt.Fprintf(os.Stderr, "Lambda execution error: %s\n", invokeResponse.Error.Message)
		os.Exit(1)
	}

	// Parse the response
	output := &wrapper.CobraLambdaOutput{}
	if err := json.Unmarshal(invokeResponse.Payload, &output); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to unmarshal response: %v\n", err)
		os.Exit(1)
	}

	// Print the output
	fmt.Print("cli", output.Stdout)

	// Send SIGTERM to the Lambda process
	fmt.Println("\nSending SIGTERM to Lambda process...")
	if err := cmd.Process.Signal(syscall.SIGKILL); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send SIGTERM: %v\n", err)
	}

	// Wait a moment for the signal to be processed
	time.Sleep(100 * time.Millisecond)

	// Try to call Function.Ping after sending SIGTERM
	fmt.Println("Attempting to call Function.Ping after SIGTERM...")
	pingResponse := &messages.PingResponse{}
	if err := client.Call("Function.Ping", messages.PingRequest{}, pingResponse); err != nil {
		fmt.Fprintf(os.Stderr, "Function.Ping failed (expected): %v\n", err)
	} else {
		fmt.Println("Function.Ping succeeded unexpectedly")
	}
}

// waitForServer polls the given port until it's available or timeout is reached
func waitForServer(port string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%s", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for server on port %s", port)
}
