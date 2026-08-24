// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

type mcpSession struct {
	workspace *MaterializedWorkspace
	ctx       context.Context
	cancel    context.CancelFunc
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	client    *client.Client
	stdout    *boundedBuffer
	stderr    *boundedBuffer
	wait      chan error
	closeOnce sync.Once
	closeErr  error
}

func startMCPSession(parent context.Context, workspace *MaterializedWorkspace) (*mcpSession, error) {
	ctx, cancel := context.WithTimeout(parent, maximumScenarioTimeout)
	command := exec.CommandContext(ctx, repokeeperPath, "--config", workspace.ConfigPath, "mcp")
	command.Dir = workspace.WorkspaceRoot
	command.Env = append([]string(nil), workspace.Env...)
	command.WaitDelay = 2 * time.Second
	configureProcessTree(command)
	stdin, err := command.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("MCP stdin pipe: %w", err)
	}
	stdoutPipe, err := command.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("MCP stdout pipe: %w", err)
	}
	stdout := newBoundedBuffer(maximumCapturedOutput)
	stderr := newBoundedBuffer(maximumCapturedOutput)
	command.Stderr = stderr
	if err := command.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start MCP process: %w", err)
	}
	if err := attachProcessTree(command); err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		cancel()
		return nil, fmt.Errorf("attach MCP process tree: %w", err)
	}
	wait := make(chan error, 1)
	go func() { wait <- command.Wait() }()

	recordedReader := io.TeeReader(stdoutPipe, stdout)
	stdio := transport.NewIO(recordedReader, stdin, io.NopCloser(strings.NewReader("")))
	mcpClient := client.NewClient(stdio)
	session := &mcpSession{workspace: workspace, ctx: ctx, cancel: cancel, cmd: command, stdin: stdin, client: mcpClient, stdout: stdout, stderr: stderr, wait: wait}
	if err := mcpClient.Start(ctx); err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("start MCP transport: %w", err)
	}
	callCtx, callCancel := context.WithTimeout(ctx, 10*time.Second)
	defer callCancel()
	_, err = mcpClient.Initialize(callCtx, mcp.InitializeRequest{Params: mcp.InitializeParams{ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION, Capabilities: mcp.ClientCapabilities{}, ClientInfo: mcp.Implementation{Name: "repokeeper-e2e", Version: "1"}}})
	if err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("initialize MCP session: %w\nstderr:\n%s", err, stderr.Bytes())
	}
	return session, nil
}

func (session *mcpSession) listTools(ctx context.Context) ([]mcp.Tool, error) {
	tools := make([]mcp.Tool, 0)
	for tool, err := range session.client.IterTools(ctx, mcp.ListToolsRequest{}) {
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

func (session *mcpSession) callTool(ctx context.Context, name string, arguments map[string]any) (*mcp.CallToolResult, error) {
	return session.client.CallTool(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Name: name, Arguments: arguments}})
}

func (session *mcpSession) Close() error {
	session.closeOnce.Do(func() {
		_ = session.stdin.Close()
		_ = session.client.Close()
		select {
		case err := <-session.wait:
			var exitError *exec.ExitError
			if err != nil && !errors.As(err, &exitError) {
				session.closeErr = fmt.Errorf("wait for MCP process: %w", err)
			}
		case <-time.After(2 * time.Second):
			_ = forceTerminateProcessTree(session.cmd)
			session.cancel()
			select {
			case <-session.wait:
			case <-time.After(2 * time.Second):
				session.closeErr = fmt.Errorf("MCP process did not terminate after forced cleanup")
			}
		}
		releaseProcessTree(session.cmd)
		session.cancel()
		if frameErr := validateJSONRPCFrames(session.stdout.Bytes()); frameErr != nil && session.closeErr == nil {
			session.closeErr = fmt.Errorf("invalid MCP stdout: %w\nstderr:\n%s", frameErr, session.stderr.Bytes())
		}
	})
	return session.closeErr
}

func validateJSONRPCFrames(data []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), maximumCapturedOutput)
	frame := 0
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		frame++
		var envelope struct {
			JSONRPC string `json:"jsonrpc"`
		}
		if err := json.Unmarshal(line, &envelope); err != nil {
			return fmt.Errorf("frame %d is not JSON: %w", frame, err)
		}
		if envelope.JSONRPC != "2.0" {
			return fmt.Errorf("frame %d has jsonrpc %q", frame, envelope.JSONRPC)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if frame == 0 {
		return fmt.Errorf("no JSON-RPC frames recorded")
	}
	return nil
}
