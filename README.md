# cobra-lambda

Run your [Cobra](https://github.com/spf13/cobra) CLI applications in AWS Lambda and invoke them remotely as if they were running locally on your machine.

## What is cobra-lambda?

cobra-lambda bridges the gap between traditional command-line applications and serverless functions. It provides:

- **A Go library (wrapper package)** - Wrap any Cobra CLI application to run in AWS Lambda with full stdout/stderr capture
- **A CLI client** - Invoke remote Lambda-hosted CLI applications from your terminal with familiar CLI arguments
- **Seamless output handling** - Captures and returns all output (stdout, stderr, and Cobra output) as structured data

Perfect for running CLI tools, automation scripts, or administrative commands serverless without maintaining long-running servers.

## Installation

### CLI Client (for invoking remote Lambda functions)

Install the `cobra-lambda` CLI tool to invoke remote Cobra applications hosted in Lambda:

```bash
go install github.com/JayJamieson/cobra-lambda@latest
```

Or build from source:

```bash
git clone https://github.com/JayJamieson/cobra-lambda.git
cd cobra-lambda
go build -o cobra-lambda .
```

### Go Library (for wrapping Cobra apps in Lambda)

Add the wrapper package to your Lambda function project:

```bash
go get github.com/JayJamieson/cobra-lambda/wrapper
```

## Quick Start

### 1. Wrap Your Cobra App for Lambda

Create a Lambda handler that wraps your Cobra CLI:

```go
package main

import (
    "context"
    "encoding/json"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/JayJamieson/cobra-lambda/wrapper"
    "github.com/spf13/cobra"
)

func Handle(ctx context.Context, event json.RawMessage) (any, error) {
    // Parse arguments from Lambda event
    var args []string
    if err := json.Unmarshal(event, &args); err != nil {
        return nil, err
    }

    // Create your Cobra command
    var name string
    cmd := &cobra.Command{
        Use: "myapp",
        Run: func(cmd *cobra.Command, args []string) {
            cmd.Printf("Hello, %s!\n", name)
        },
    }
    cmd.Flags().StringVar(&name, "name", "World", "Name to greet")

    // Wrap and execute
    w := wrapper.NewCobraLambda(cmd)
    output, err := w.Execute(args)
    if err != nil {
        return nil, err
    }

    // Return structured output
    return map[string]any{
        "stdout": output.Output,
    }, nil
}

func main() {
    lambda.Start(Handle)
}
```

Build and deploy to Lambda:

```bash
GOOS=linux GOARCH=amd64 go build -o bootstrap main.go
zip function.zip bootstrap
# Deploy to AWS Lambda with runtime: provided.al2
```

### 2. Invoke from Your Terminal

Use the CLI client to invoke your Lambda-hosted Cobra app:

```bash
# Basic invocation
cobra-lambda --name my-lambda-function

# With arguments and flags
cobra-lambda --name my-lambda-function --name Alice

# Pass any Cobra CLI arguments
cobra-lambda --name my-lambda-function subcommand --flag value arg1 arg2
```

The CLI forwards all arguments after `--name [function-name]` to your Lambda function.

## Usage Examples

### Basic Command

```go
package main

import (
    "github.com/JayJamieson/cobra-lambda/wrapper"
    "github.com/spf13/cobra"
)

func example() {
    cmd := &cobra.Command{
        Use: "greet",
        Run: func(cmd *cobra.Command, args []string) {
            cmd.Println("Hello from Cobra!")
        },
    }

    w := wrapper.NewCobraLambda(context.TODO(), cmd)
    output, err := w.Execute([]string{})
    if err != nil {
        panic(err)
    }

    // output.Output contains all captured output
}
```

### With Subcommands

```go
rootCmd := &cobra.Command{Use: "root"}
subCmd := &cobra.Command{
    Use: "deploy",
    Run: func(cmd *cobra.Command, args []string) {
        cmd.Println("Deploying...")
    },
}
rootCmd.AddCommand(subCmd)

w := wrapper.NewCobraLambda(context.TODO(), rootCmd)
output, err := w.Execute([]string{"deploy"})
```

## Output Structure

The `OutputCapture` struct contains all captured output in a single field:

```go
type OutputCapture struct {
    // Output contains all captured output from Cobra, stdout, and stderr
    Output string
}
```

All output streams (Cobra's SetOut/SetErr, os.Stdout, and os.Stderr) are captured into a single shared buffer, preserving the order of output as it was written.

## Thread Safety

The wrapper is thread-safe:
- Each `CobraWrapper` instance uses a mutex to serialize `Execute()` calls
- Internal buffers are thread-safe with their own locks
- os.Stdout/os.Stderr are restored after each execution, even on panic

For concurrent executions, create separate wrapper instances per goroutine, or reuse a single instance (executions will be serialized automatically).

## CLI Client Usage

The `cobra-lambda` CLI tool invokes remote Lambda functions:

```bash
cobra-lambda --name <function-name> [cobra-args...]
```

### Examples

```bash
# Simple invocation
cobra-lambda --name my-cli-app

# With flags
cobra-lambda --name my-cli-app --verbose --output json

# With subcommands
cobra-lambda --name my-cli-app deploy --environment prod

# Help
cobra-lambda --help
```

### AWS Configuration

The CLI uses the AWS SDK for Go v2 and respects standard AWS configuration:

- AWS credentials from `~/.aws/credentials` or environment variables
- Region from `AWS_REGION` environment variable or AWS config
- IAM permissions required: `lambda:InvokeFunction`

## Testing

Run the wrapper package test suite:

```bash
cd wrapper
go test -v
```

## How It Works

1. **Client Side**: The `cobra-lambda` CLI tool extracts the `--name` flag to identify the target Lambda function, then forwards all remaining arguments as a JSON array payload.

2. **Lambda Side**: The wrapper package:
   - Intercepts `os.Stdout` and `os.Stderr` using pipes
   - Redirects Cobra's output streams to a shared buffer
   - Unmarshals the JSON array of arguments
   - Executes the Cobra command with the provided arguments
   - Returns all captured output as structured data

3. **Response**: The client displays the returned output, making the remote execution feel like a local CLI invocation.

## Examples in This Repository

See the `iac/` directory for complete examples:

- `iac/go-demo/` - Go-based Cobra CLI wrapped for Lambda
- `iac/node-demo/` - Node.js implementation showing the same pattern
- `iac/main.tf` - Terraform configuration for deploying Lambda functions

## License

MIT
