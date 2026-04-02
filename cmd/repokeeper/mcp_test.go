// SPDX-License-Identifier: MIT
package repokeeper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/obs"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
)

func TestBuildMCPLoggerWithoutPathReturnsNopLogger(t *testing.T) {
	t.Parallel()

	logger, cleanup, err := buildMCPLogger("")
	if err != nil {
		t.Fatalf("buildMCPLogger returned error: %v", err)
	}
	if logger == nil {
		t.Fatal("expected logger")
	}
	if cleanup == nil {
		t.Fatal("expected cleanup function")
	}

	logger.Infof("info %d", 1)
	logger.Debugf("debug %d", 2)
	logger.Warnf("warn %d", 3)
	cleanup()
}

func TestBuildMCPLoggerWritesFormattedLines(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "repokeeper-mcp.log")
	logger, cleanup, err := buildMCPLogger(path)
	if err != nil {
		t.Fatalf("buildMCPLogger returned error: %v", err)
	}
	if logger == nil {
		t.Fatal("expected logger")
	}
	if cleanup == nil {
		t.Fatal("expected cleanup function")
	}

	logger.Infof("server start %d", 1)
	logger.Debugf("debug state %s", "ready")
	logger.Warnf("warn state %s", "degraded")
	cleanup()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	text := string(contents)
	for _, want := range []string{
		"[INFO]  server start 1",
		"[DEBUG] debug state ready",
		"[WARN]  warn state degraded",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("log file %q missing %q", text, want)
		}
	}
}

func TestBuildMCPLoggerReturnsOpenFileErrors(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing", "repokeeper-mcp.log")
	logger, cleanup, err := buildMCPLogger(path)
	if err == nil {
		t.Fatal("expected error for missing parent directory")
	}
	if logger != nil {
		t.Fatalf("expected nil logger on error, got %#v", logger)
	}
	if cleanup != nil {
		t.Fatal("expected nil cleanup on error")
	}
	if !strings.Contains(err.Error(), "open log file") {
		t.Fatalf("error %q does not mention log file open failure", err)
	}
}

func TestNewMCPEngineWiresLoggerIntoGitAdapter(t *testing.T) {
	t.Parallel()

	logger := obs.NopLogger()
	eng := newMCPEngine(&config.Config{Registry: &registry.Registry{}}, logger)

	adapter, ok := eng.Adapter().(*vcs.GitAdapter)
	if !ok {
		t.Fatalf("expected *vcs.GitAdapter, got %T", eng.Adapter())
	}
	if adapter.Logger != logger {
		t.Fatal("expected MCP logger to be wired into git adapter")
	}
}
