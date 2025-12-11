package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

type RunMode int

const (
	// ModeBinary runs a compiled Lambda binary
	ModeBinary RunMode = iota
	// ModeGoRun uses 'go run' to execute the Lambda function
	ModeGoRun
)

type Runner struct {
	Mode       RunMode
	Debug      bool
	ServerPort string
}

type CommandConfig struct {
	LambdaPath string
	LambdaArgs []string
}

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

	if len(args) < 1 {
		return nil, fmt.Errorf("missing lambda path argument")
	}

	switch r.Mode {
	case ModeBinary:

		config.LambdaPath = args[0]
		if len(args) > 1 {
			config.LambdaArgs = args[1:]
		}

	case ModeGoRun:
		isGoFile := strings.HasSuffix(args[0], ".go")

		if !isGoFile {
			return nil, fmt.Errorf("invalid go file: %s", args[0])
		}

		config.LambdaPath = args[0]
		if len(args) > 1 {
			config.LambdaArgs = args[1:]
		}

	default:
		return nil, fmt.Errorf("unknown run mode: %d", r.Mode)
	}

	return config, nil
}

func (r *Runner) CreateCommand(config *CommandConfig) (*exec.Cmd, error) {
	var cmd *exec.Cmd

	if _, err := os.Stat(config.LambdaPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("not found at %s", config.LambdaPath)
	}

	switch r.Mode {
	case ModeBinary:
		cmd = exec.Command(config.LambdaPath, config.LambdaArgs...)

	case ModeGoRun:
		// Construct args: go run <lambda-path> [lambda-args...]
		args := append([]string{"run", config.LambdaPath}, config.LambdaArgs...)
		cmd = exec.Command("go", args...)
	default:
		return nil, fmt.Errorf("unknown run mode: %d", r.Mode)
	}

	// needed in order to properly kill lambda process when run with go run
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	cmd.Env = append(os.Environ(), fmt.Sprintf("_LAMBDA_SERVER_PORT=%s", r.ServerPort))

	return cmd, nil
}

func (r *Runner) KillProcessGroup(cmd *exec.Cmd, signal syscall.Signal) error {
	if cmd.Process == nil {
		return fmt.Errorf("process not started")
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		return fmt.Errorf("failed to get process group ID: %w", err)
	}

	if err := syscall.Kill(-pgid, signal); err != nil {
		return fmt.Errorf("failed to send signal %v: %w", signal, err)
	}

	return nil
}

func (r *Runner) Debugf(format string, args ...any) {
	if r.Debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}
