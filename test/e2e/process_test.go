// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	maximumScenarioTimeout = 30 * time.Second
	maximumCapturedOutput  = 1 << 20
)

type ExecutionResult struct {
	Operation   string
	Executable  string
	Arguments   []string
	WorkingDir  string
	Root        string
	ConfigPath  string
	ExitCode    int
	Stdout      []byte
	Stderr      []byte
	Duration    time.Duration
	TimedOut    bool
	LaunchError error
}

func (result ExecutionResult) Diagnostics() string {
	exit := fmt.Sprintf("%d", result.ExitCode)
	if result.LaunchError != nil {
		exit = "launch error: " + result.LaunchError.Error()
	}
	return fmt.Sprintf(
		"operation: %s\ncommand: %q %q\nworking directory: %s\nscenario root: %s\nconfig path: %s\nexit: %s\ntimed out: %t\nduration: %s\nstdout:\n%s\nstderr:\n%s",
		result.Operation,
		result.Executable,
		result.Arguments,
		result.WorkingDir,
		result.Root,
		result.ConfigPath,
		exit,
		result.TimedOut,
		result.Duration,
		result.Stdout,
		result.Stderr,
	)
}

func runCommand(parent context.Context, timeout time.Duration, operation, executable string, arguments []string, dir string, environment []string, root, configPath string) ExecutionResult {
	if timeout <= 0 || timeout > maximumScenarioTimeout {
		timeout = maximumScenarioTimeout
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	stdout := newBoundedBuffer(maximumCapturedOutput)
	stderr := newBoundedBuffer(maximumCapturedOutput)
	cmd := exec.CommandContext(ctx, executable, arguments...)
	cmd.Dir = dir
	cmd.Env = append([]string(nil), environment...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.WaitDelay = 2 * time.Second
	configureProcessTree(cmd)

	result := ExecutionResult{
		Operation: operation, Executable: executable, Arguments: append([]string(nil), arguments...),
		WorkingDir: dir, Root: root, ConfigPath: configPath, ExitCode: -1,
	}
	started := time.Now()
	err := cmd.Start()
	if err != nil {
		result.Duration = time.Since(started)
		result.Stdout = stdout.Bytes()
		result.Stderr = stderr.Bytes()
		result.LaunchError = err
		return result
	}
	if err := attachProcessTree(cmd); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		result.Duration = time.Since(started)
		result.Stdout = stdout.Bytes()
		result.Stderr = stderr.Bytes()
		result.LaunchError = fmt.Errorf("attach process tree: %w", err)
		return result
	}
	processDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			select {
			case <-processDone:
				return
			case <-time.After(500 * time.Millisecond):
				_ = forceTerminateProcessTree(cmd)
			}
		case <-processDone:
		}
	}()
	err = cmd.Wait()
	close(processDone)
	releaseProcessTree(cmd)
	result.Duration = time.Since(started)
	result.Stdout = stdout.Bytes()
	result.Stderr = stderr.Bytes()
	result.TimedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
	if err == nil {
		result.ExitCode = 0
		return result
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		result.ExitCode = exitError.ExitCode()
		return result
	}
	result.LaunchError = err
	return result
}

type boundedBuffer struct {
	mu        sync.Mutex
	buffer    bytes.Buffer
	remaining int
	truncated bool
}

func newBoundedBuffer(limit int) *boundedBuffer {
	return &boundedBuffer{remaining: limit}
}

func (buffer *boundedBuffer) Write(data []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	original := len(data)
	if buffer.remaining <= 0 {
		buffer.truncated = true
		return original, nil
	}
	if len(data) > buffer.remaining {
		data = data[:buffer.remaining]
		buffer.truncated = true
	}
	_, _ = buffer.buffer.Write(data)
	buffer.remaining -= len(data)
	return original, nil
}

func (buffer *boundedBuffer) Bytes() []byte {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	data := append([]byte(nil), buffer.buffer.Bytes()...)
	if buffer.truncated {
		data = append(data, []byte("\n<output truncated>\n")...)
	}
	return data
}

func copyBounded(destination *boundedBuffer, source io.Reader, done chan<- error) {
	_, err := io.Copy(destination, source)
	done <- err
}

func expectExit(result ExecutionResult, allowed ...int) error {
	if result.LaunchError != nil || result.TimedOut {
		return fmt.Errorf("command infrastructure failure\n%s", result.Diagnostics())
	}
	for _, code := range allowed {
		if result.ExitCode == code {
			return nil
		}
	}
	return fmt.Errorf("unexpected exit code %d (wanted %v)\n%s", result.ExitCode, allowed, result.Diagnostics())
}

func fixedOutput(value []byte) string {
	return strings.TrimSpace(string(value))
}
