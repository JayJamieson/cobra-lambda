package cli

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// RunMode represents how the Lambda function should be executed
type RunMode int

const (
	// ModeBinary runs a compiled Lambda binary
	ModeBinary RunMode = iota
	// ModeGoRun uses 'go run' to execute the Lambda function
	ModeGoRun
)

// Runner handles the execution of Lambda functions
type Runner struct {
	Mode       RunMode
	Debug      bool
	ServerPort string
}

// CommandConfig contains the parsed command configuration
type CommandConfig struct {
	LambdaPath string
	LambdaArgs []string
}

// NewRunner creates a new Runner with the specified mode
func NewRunner(mode RunMode, debug bool, serverPort string) *Runner {
	return &Runner{
		Mode:       mode,
		Debug:      debug,
		ServerPort: serverPort,
	}
}

// ParseArgs parses remaining arguments after flag parsing based on the run mode
// args should be the result of flag.Args() after flag.Parse()
func (r *Runner) ParseArgs(args []string) (*CommandConfig, error) {
	config := &CommandConfig{}
	r.Debugf("Parsing arguments: %v", args)
	switch r.Mode {
	case ModeBinary:
		if len(args) < 1 {
			return nil, fmt.Errorf("missing lambda path argument")
		}
		config.LambdaPath = args[0]
		if len(args) > 1 {
			config.LambdaArgs = args[1:]
		}

	case ModeGoRun:
		// For go run mode, we expect: -- lambda-path [args...]
		if len(args) < 2 {
			return nil, fmt.Errorf("missing lambda path argument (expected after '--')")
		}
		// if args[0] != "--" {
		// 	return nil, fmt.Errorf("expected '--' separator before lambda path, got: %s", args[0])
		// }
		config.LambdaPath = args[0]
		if len(args) > 2 {
			config.LambdaArgs = args[2:]
		}

	default:
		return nil, fmt.Errorf("unknown run mode: %d", r.Mode)
	}

	return config, nil
}

// CreateCommand creates an exec.Command based on the run mode
func (r *Runner) CreateCommand(config *CommandConfig) (*exec.Cmd, error) {
	var cmd *exec.Cmd

	switch r.Mode {
	case ModeBinary:
		// Check if lambda binary exists
		if _, err := os.Stat(config.LambdaPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("lambda binary not found at %s", config.LambdaPath)
		}
		cmd = exec.Command(config.LambdaPath, config.LambdaArgs...)

	case ModeGoRun:
		// Check if lambda source file exists
		if _, err := os.Stat(config.LambdaPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("lambda source file not found at %s", config.LambdaPath)
		}
		// Construct args: go run <lambda-path> [lambda-args...]
		args := append([]string{"run", config.LambdaPath}, config.LambdaArgs...)
		cmd = exec.Command("go", args...)

	default:
		return nil, fmt.Errorf("unknown run mode: %d", r.Mode)
	}

	// Set up process group for proper signal handling
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Add the Lambda server port to environment
	cmd.Env = append(os.Environ(), fmt.Sprintf("_LAMBDA_SERVER_PORT=%s", r.ServerPort))

	// Only show stdout/stderr if debug mode is enabled
	if r.Debug {
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = nil
		cmd.Stderr = nil
	}

	return cmd, nil
}

// KillProcessGroup kills the entire process group
func (r *Runner) KillProcessGroup(cmd *exec.Cmd, signal syscall.Signal) error {
	if cmd.Process == nil {
		return fmt.Errorf("process not started")
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		return fmt.Errorf("failed to get process group ID: %w", err)
	}

	// Kill the entire process group (note the negative sign)
	if err := syscall.Kill(-pgid, signal); err != nil {
		return fmt.Errorf("failed to send signal %v: %w", signal, err)
	}

	return nil
}

// Debugf prints debug messages if debug mode is enabled
func (r *Runner) Debugf(format string, args ...interface{}) {
	if r.Debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}
