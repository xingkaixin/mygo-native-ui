//go:build darwin || linux

package main

import (
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

// startPTY starts cmd in a new session whose controlling terminal is a
// pseudo-terminal of cols and rows, and returns the terminal's side of
// it, which reads what cmd writes and writes what it reads.
func startPTY(cmd *exec.Cmd, cols, rows int) (*os.File, error) {
	pty, tty, err := openPTY()
	if err != nil {
		return nil, err
	}
	defer tty.Close()
	if err := setSize(pty, cols, rows); err != nil {
		pty.Close()
		return nil, err
	}
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = tty, tty, tty
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true}
	if err := cmd.Start(); err != nil {
		pty.Close()
		return nil, err
	}
	go cmd.Wait()
	return pty, nil
}

// setSize sets the size of the pseudo-terminal, which signals its
// program to draw again.
func setSize(pty *os.File, cols, rows int) error {
	ws := struct{ rows, cols, x, y uint16 }{rows: uint16(rows), cols: uint16(cols)}
	return ioctl(pty, syscall.TIOCSWINSZ, uintptr(unsafe.Pointer(&ws)))
}

func ioctl(f *os.File, req, arg uintptr) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), req, arg); errno != 0 {
		return errno
	}
	return nil
}
