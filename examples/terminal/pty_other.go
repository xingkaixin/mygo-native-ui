//go:build !darwin && !linux

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func init() {
	fmt.Fprintln(os.Stderr, "terminal: the example opens a pseudo-terminal on macOS and Linux only")
	os.Exit(1)
}

func startPTY(*exec.Cmd, int, int) (*os.File, error) { return nil, errors.ErrUnsupported }

func setSize(*os.File, int, int) error { return errors.ErrUnsupported }
