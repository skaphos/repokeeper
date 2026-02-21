// SPDX-License-Identifier: MIT
// Package obs provides observability primitives (logging) for RepoKeeper internals.
package obs

// Logger provides structured diagnostic output for engine operations.
type Logger interface {
	Infof(format string, args ...any)
	Debugf(format string, args ...any)
	Warnf(format string, args ...any)
}

// NopLogger returns a Logger that discards all output.
func NopLogger() Logger { return nopLogger{} }

type nopLogger struct{}

func (nopLogger) Infof(string, ...any)  {}
func (nopLogger) Debugf(string, ...any) {}
func (nopLogger) Warnf(string, ...any)  {}
