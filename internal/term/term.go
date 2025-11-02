package term

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// InjectCommandToTTY injects the given command into the controlling terminal's
// input buffer so it appears as if the user typed it. If pressEnter is true,
// a trailing newline is also injected to execute the command immediately
func InjectCommandToTTY(command string, pressEnter bool) error {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("unable to open /dev/tty: %w", err)
	}
	defer tty.Close()

	fd := tty.Fd()

	injectByte := func(b byte) error {
		// Use ioctl TIOCSTI to stuff a byte into the terminal input buffer
		// This makes it appear as if the user typed the byte at the prompt
		_, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(unix.TIOCSTI), uintptr(unsafe.Pointer(&b)))
		if errno != 0 {
			return errno
		}
		return nil
	}

	for i := 0; i < len(command); i++ {
		if err := injectByte(command[i]); err != nil {
			return fmt.Errorf("failed to inject input: %w", err)
		}
	}

	if pressEnter {
		if err := injectByte('\n'); err != nil {
			return fmt.Errorf("failed to inject enter: %w", err)
		}
	}

	return nil
}
