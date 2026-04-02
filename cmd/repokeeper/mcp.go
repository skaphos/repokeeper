// SPDX-License-Identifier: MIT
package repokeeper

import (
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/mcpserver"
	"github.com/skaphos/repokeeper/internal/obs"
	"github.com/skaphos/repokeeper/internal/vcs"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the MCP (Model Context Protocol) server for agent-native querying",
	Long:  "Exposes RepoKeeper operations as structured MCP tools over stdio transport. Designed for integration with agent runtimes (Claude Code, OpenCode, Codex).",
	RunE: func(cmd *cobra.Command, _ []string) error {
		logFile, _ := cmd.Flags().GetString("log-file")
		logger, cleanup, err := buildMCPLogger(logFile)
		if err != nil {
			return err
		}
		defer cleanup()

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		cfgPath, err := config.ResolveConfigPath(configOverride(cmd), cwd)
		if err != nil {
			return err
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		if cfg.Registry == nil {
			return fmt.Errorf("registry not found in %q (run repokeeper scan first)", cfgPath)
		}

		eng := newMCPEngine(cfg, logger)
		srv := mcpserver.New(eng, cfgPath, Version, logger)

		logger.Infof("MCP server starting (config=%s, repos=%d)", cfgPath, len(cfg.Registry.Entries))
		return server.ServeStdio(srv.Inner(), server.WithErrorLogger(log.New(os.Stderr, "mcp: ", log.LstdFlags)))
	},
}

func init() {
	mcpCmd.Flags().String("log-file", "", "debug log file path (stdout is owned by MCP protocol in stdio mode)")
	rootCmd.AddCommand(mcpCmd)
}

// buildMCPLogger returns a logger that writes to the specified file, or a
// nop logger if no file is specified. In stdio mode, stdout is owned by the
// MCP protocol so diagnostic output must go to a file.
func buildMCPLogger(path string) (obs.Logger, func(), error) {
	if path == "" {
		return obs.NopLogger(), func() {}, nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file: %w", err)
	}
	l := &fileLogger{out: log.New(f, "", log.LstdFlags)}
	return l, func() { _ = f.Close() }, nil
}

func newMCPEngine(cfg *config.Config, logger obs.Logger) *engine.Engine {
	adapter := vcs.NewGitAdapter(nil, logger)
	return engine.New(cfg, cfg.Registry, adapter, vcs.NewGitErrorClassifier(), vcs.NewGitURLNormalizer(), logger)
}

type fileLogger struct{ out *log.Logger }

func (l *fileLogger) Infof(format string, args ...any)  { l.out.Printf("[INFO]  "+format, args...) }
func (l *fileLogger) Debugf(format string, args ...any) { l.out.Printf("[DEBUG] "+format, args...) }
func (l *fileLogger) Warnf(format string, args ...any)  { l.out.Printf("[WARN]  "+format, args...) }
