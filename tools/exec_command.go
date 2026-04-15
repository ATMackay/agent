package tools

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

const (
	defaultExecTimeoutSeconds = 30
	maxExecTimeoutSeconds     = 300
	maxExecOutputBytes        = 64 * 1024 // 64 KB per stream
)

// ExecCommandArgs are the inputs to the exec_command tool.
type ExecCommandArgs struct {
	Command        string   `json:"command"`
	Args           []string `json:"args,omitempty"`
	WorkDir        string   `json:"work_dir,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
}

// ExecCommandResult is returned by the exec_command tool.
type ExecCommandResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	TimedOut bool   `json:"timed_out,omitempty"`
}

// NewExecCommandTool returns an exec_command function tool.
func NewExecCommandTool() (tool.Tool, error) {
	t, err := functiontool.New(
		functiontool.Config{
			Name: "exec_command",
			Description: "Execute a command and return its stdout, stderr, and exit code. " +
				"Use for running scripts, building code, extracting text from documents (pdftotext, pandoc), " +
				"or any other CLI task. work_dir defaults to the session work directory.",
		},
		newExecCommandTool(),
	)
	if err != nil {
		return nil, fmt.Errorf("create exec_command tool: %w", err)
	}
	return t, nil
}

func newExecCommandTool() func(tool.Context, ExecCommandArgs) (ExecCommandResult, error) {
	return func(ctx tool.Context, args ExecCommandArgs) (ExecCommandResult, error) {
		slog.Info("tool call", "function", "exec_command", "command", args.Command, "args", args.Args)

		if strings.TrimSpace(args.Command) == "" {
			return ExecCommandResult{}, fmt.Errorf("command is required")
		}

		timeout := args.TimeoutSeconds
		if timeout <= 0 {
			timeout = defaultExecTimeoutSeconds
		}
		if timeout > maxExecTimeoutSeconds {
			timeout = maxExecTimeoutSeconds
		}

		workDir := args.WorkDir
		if workDir == "" {
			workDir = getWorkDir(ctx)
		}

		cmdCtx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()

		cmd := exec.CommandContext(cmdCtx, args.Command, args.Args...)
		if workDir != "" {
			cmd.Dir = workDir
		}

		var stdout, stderr limitedBuffer
		stdout.max = maxExecOutputBytes
		stderr.max = maxExecOutputBytes
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		runErr := cmd.Run()

		timedOut := cmdCtx.Err() == context.DeadlineExceeded
		exitCode := 0

		if runErr != nil {
			var exitErr *exec.ExitError
			if ok := isExitError(runErr, &exitErr); ok {
				exitCode = exitErr.ExitCode()
			} else if !timedOut {
				return ExecCommandResult{}, fmt.Errorf("exec %q: %w", args.Command, runErr)
			}
		}

		return ExecCommandResult{
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			ExitCode: exitCode,
			TimedOut: timedOut,
		}, nil
	}
}

// limitedBuffer caps writes at max bytes, discarding the rest.
type limitedBuffer struct {
	buf bytes.Buffer
	max int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.max - b.buf.Len()
	if remaining <= 0 {
		return len(p), nil
	}
	if len(p) > remaining {
		p = p[:remaining]
	}
	n, err := b.buf.Write(p)
	return n, err
}

func (b *limitedBuffer) String() string { return b.buf.String() }

func isExitError(err error, target **exec.ExitError) bool {
	var exitErr *exec.ExitError
	if e, ok := err.(*exec.ExitError); ok {
		*target = e
		return true
	}
	_ = exitErr
	return false
}
