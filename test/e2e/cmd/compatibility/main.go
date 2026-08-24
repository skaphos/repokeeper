// SPDX-License-Identifier: MIT
//go:build integration

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/skaphos/repokeeper/test/e2e/internal/compatibility"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("operation is required: matrix, provision, verify-version, write-evidence, verify-evidence, or verify-docs")
	}
	root, err := moduleRoot()
	if err != nil {
		return err
	}
	declarationPath := filepath.Join(root, "test", "e2e", "testdata", "git-compatibility.json")
	declaration, err := compatibility.Load(declarationPath)
	if err != nil {
		return err
	}
	switch arguments[0] {
	case "matrix":
		flags := flag.NewFlagSet("matrix", flag.ContinueOnError)
		flags.SetOutput(os.Stderr)
		scope := flags.String("scope", "routine", "routine or release")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		result, err := compatibility.Expand(declaration, *scope)
		if err != nil {
			return err
		}
		return emitJSON(result)
	case "provision":
		flags := flag.NewFlagSet("provision", flag.ContinueOnError)
		flags.SetOutput(os.Stderr)
		key := flags.String("cell", "", "cell key")
		prefix := flags.String("prefix", "", "test-owned install prefix")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		if *prefix == "" {
			return errors.New("--prefix is required")
		}
		cell, err := compatibility.FindCell(declaration, *key)
		if err != nil {
			return err
		}
		result, err := compatibility.Provision(context.Background(), cell, *prefix)
		if err != nil {
			return err
		}
		return emitJSON(result)
	case "verify-version":
		flags := flag.NewFlagSet("verify-version", flag.ContinueOnError)
		flags.SetOutput(os.Stderr)
		key := flags.String("cell", "", "cell key")
		gitPath := flags.String("git", "", "Git executable path")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		if *gitPath == "" {
			return errors.New("--git is required")
		}
		cell, err := compatibility.FindCell(declaration, *key)
		if err != nil {
			return err
		}
		result, err := compatibility.VerifyVersion(context.Background(), cell, *gitPath)
		if err != nil {
			return err
		}
		return emitJSON(result)
	case "write-evidence":
		return writeEvidence(declaration, arguments[1:])
	case "verify-evidence":
		return verifyEvidence(declaration, arguments[1:])
	case "verify-docs":
		design, err := os.ReadFile(filepath.Join(root, "DESIGN.md"))
		if err != nil {
			return err
		}
		if err := compatibility.VerifyDocs(declaration, design); err != nil {
			return err
		}
		return emitJSON(map[string]any{"valid": true, "supported_minors": declaration.SupportedMinors, "cells": len(declaration.Cells)})
	default:
		return fmt.Errorf("unknown operation %q", arguments[0])
	}
}

func writeEvidence(declaration compatibility.Declaration, arguments []string) error {
	flags := flag.NewFlagSet("write-evidence", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	key := flags.String("cell", "", "cell key")
	output := flags.String("output", "", "evidence output path")
	tag := flags.String("tag", "", "immutable candidate tag")
	commit := flags.String("commit", "", "source commit")
	actual := flags.String("actual-git", "", "raw git --version output")
	status := flags.String("status", "passed", "passed, failed, or skipped")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *output == "" || !strings.HasPrefix(*tag, "v") || *commit == "" {
		return errors.New("--output, v-prefixed --tag, and --commit are required")
	}
	cell, err := compatibility.FindCell(declaration, *key)
	if err != nil {
		return err
	}
	digests := map[string]string{"provisioner": cell.Provisioner.SHA256}
	if cell.RootFS != nil {
		digests["rootfs"] = cell.RootFS.SHA256
		for _, prerequisite := range cell.RootFS.BuildPrerequisites {
			digests["package:"+prerequisite.Name] = prerequisite.Version
		}
	}
	result, err := compatibility.WriteEvidence(*output, compatibility.CompatibilityResult{CellKey: *key, CandidateTag: *tag, Commit: *commit, Environment: cell.Environment, RunnerLabel: cell.RunnerLabel, GitMinor: cell.GitMinor, ExpectedGit: cell.GitPatch, ActualGit: *actual, InputDigests: digests, Status: *status})
	if err != nil {
		return err
	}
	return emitJSON(result)
}

func verifyEvidence(declaration compatibility.Declaration, arguments []string) error {
	flags := flag.NewFlagSet("verify-evidence", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	directory := flags.String("directory", "", "evidence directory")
	tag := flags.String("tag", "", "candidate tag")
	commit := flags.String("commit", "", "source commit")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *directory == "" || *tag == "" || *commit == "" {
		return errors.New("--directory, --tag, and --commit are required")
	}
	summary, err := compatibility.VerifyEvidence(declaration, *directory, *tag, *commit)
	if err != nil {
		return fmt.Errorf("%w\nsummary=%s", err, mustJSON(summary))
	}
	return emitJSON(summary)
}

func emitJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}
func mustJSON(value any) string { data, _ := json.Marshal(value); return string(data) }
func moduleRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("resolve compatibility command source")
	}
	for dir := filepath.Dir(file); ; dir = filepath.Dir(dir) {
		if info, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && info.Mode().IsRegular() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found")
		}
	}
}
