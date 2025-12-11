package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"syscall"
	"time"

	"github.com/JayJamieson/cobra-lambda/cli"
	"github.com/JayJamieson/cobra-lambda/wrapper"
	"github.com/aws/aws-lambda-go/lambda/messages"
)

const (
	lambdaServerPort = "8001"
	helpMessage      = `Usage: rpc [flags] [lambda-path] [args...]
       rpc [flags] -- [lambda-path] [args...]

Runs a Go Lambda function locally over RPC.

Flags:
  --debug         Enable debug logging
  --go-run        Use 'go run' instead of compiled binary (requires '--' separator)

Arguments:
  lambda-path     Path to the compiled Lambda binary or source file
  args...         Arguments to pass to the Lambda function

Examples:
  # Run compiled binary
  rpc ./lambda-binary arg1 arg2

  # Run with go run
  rpc --go-run -- cmd/lambda/main.go arg1 arg2

  # Debug mode
  rpc --debug ./lambda-binary

The Lambda binary will be started with _LAMBDA_SERVER_PORT=8001 and invoked over RPC.
`
)

var (
	debugFlag = flag.Bool("debug", false, "Enable debug logging")
	goRunFlag = flag.Bool("go-run", false, "Use 'go run' instead of compiled binary")
)

func main() {
	flag.Usage = func() {
		fmt.Print(helpMessage)
	}
	flag.Parse()

	// Determine run mode
	mode := cli.ModeBinary
	if *goRunFlag {
		mode = cli.ModeGoRun
	}

	// Create runner
	runner := cli.NewRunner(mode, *debugFlag, lambdaServerPort)

	// Parse arguments based on mode (use flag.Args() which contains non-flag arguments)
	config, err := runner.ParseArgs(flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	runner.Debugf("Mode: %v", mode)
	runner.Debugf("Lambda path: %s", config.LambdaPath)
	runner.Debugf("Lambda args: %v", config.LambdaArgs)

	// Create the command
	cmd, err := runner.CreateCommand(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start Lambda process: %v\n", err)
		os.Exit(1)
	}

	runner.Debugf("Lambda process started with PID: %d", cmd.Process.Pid)

	// Ensure we kill the process on exit
	defer func() {
		if cmd.Process != nil {
			runner.Debugf("Cleaning up Lambda process...")
			_ = runner.KillProcessGroup(cmd, syscall.SIGKILL)
			_ = cmd.Wait()
		}
	}()

	// Wait for the Lambda server to be ready
	if err := waitForServer(lambdaServerPort, 5*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "Lambda server failed to start: %v\n", err)
		os.Exit(1)
	}

	runner.Debugf("Lambda server is ready on port %s", lambdaServerPort)

	// Connect to the Lambda RPC server
	client, err := rpc.Dial("tcp", fmt.Sprintf("localhost:%s", lambdaServerPort))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to Lambda server: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	runner.Debugf("Connected to Lambda RPC server")

	// Prepare the invocation request
	argsEvent := wrapper.CobraLambdaEvent{Args: config.LambdaArgs}
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
	runner.Debugf("Invoking Lambda function...")
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
	fmt.Print(output.Stdout)

	// Send SIGTERM to the Lambda process
	runner.Debugf("Sending SIGTERM to Lambda process...")
	if err := runner.KillProcessGroup(cmd, syscall.SIGTERM); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send SIGTERM: %v\n", err)
	}

	// Wait a moment for the signal to be processed
	time.Sleep(500 * time.Millisecond)

	// Try to call Function.Ping after sending SIGTERM
	runner.Debugf("Attempting to call Function.Ping after SIGTERM...")
	pingResponse := &messages.PingResponse{}

	if err := client.Call("Function.Ping", messages.PingRequest{}, pingResponse); err != nil {
		runner.Debugf("Function.Ping failed (expected): %v", err)
	} else {
		runner.Debugf("Function.Ping succeeded unexpectedly")
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
