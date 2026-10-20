package main

import (
	"bytes"
	"os"
	"syscall"
	"unsafe"
)

// openPTY opens a pseudo-terminal: its terminal's side, and the side the
// program takes as its terminal.
func openPTY() (pty, tty *os.File, err error) {
	pty, err = os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	var name [128]byte
	if err = ioctl(pty, syscall.TIOCPTYGRANT, 0); err == nil {
		err = ioctl(pty, syscall.TIOCPTYUNLK, 0)
	}
	if err == nil {
		err = ioctl(pty, syscall.TIOCPTYGNAME, uintptr(unsafe.Pointer(&name)))
	}
	if err == nil {
		path, _, _ := bytes.Cut(name[:], []byte{0})
		tty, err = os.OpenFile(string(path), os.O_RDWR|syscall.O_NOCTTY, 0)
	}
	if err != nil {
		pty.Close()
		return nil, nil, err
	}
	return pty, tty, nil
}
