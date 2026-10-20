package main

import (
	"os"
	"strconv"
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
	var unlock int32
	var n uint32
	if err = ioctl(pty, syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock))); err == nil {
		err = ioctl(pty, syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n)))
	}
	if err == nil {
		tty, err = os.OpenFile("/dev/pts/"+strconv.Itoa(int(n)), os.O_RDWR|syscall.O_NOCTTY, 0)
	}
	if err != nil {
		pty.Close()
		return nil, nil, err
	}
	return pty, tty, nil
}
