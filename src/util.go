/**
 * @file util.go
 * @author Vladyslav Aviedov <vladaviedov at protonmail dot com>
 * @version pre0.8.0
 * @date 2026
 * @license GPLv3.0
 * @brief Utility functions.
 */
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func convertPath(path string, fileDir string) string {
	// Absolute paths
	if filepath.IsAbs(path) {
		return path
	}

	// Root path substitution
	if path[0] == '$' {
		return filepath.Join(opts.Root, path[1:])
	}

	// Relative paths
	return filepath.Join(fileDir, path)
}

func runSystemCommand(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run command: %w", err)
	}

	return string(out), nil
}

func pandocConvert(inputPath string, inputFormat string) (*os.File, error) {
	// Create temp file for pandoc output
	pandocOutput, err := os.CreateTemp("", "*.shpp")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	// Setup pandoc command
	args := []string{"-t", "html", "--quiet", "-o", pandocOutput.Name()}
	if inputFormat != "" {
		args = append(args, "-f", inputFormat)
	}
	if opts.PandocOptsFile != "" {
		args = append(args, "-d", opts.PandocOptsFile)
	}
	args = append(args, inputPath)

	// Run pandoc redirecting stderr to here
	cmd := exec.Command(opts.PandocBinary, args...)
	cmd.Stderr = os.Stderr
	_, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute pandoc: %w", err)
	}

	return pandocOutput, nil
}
