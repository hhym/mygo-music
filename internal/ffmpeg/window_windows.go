//go:build windows

package ffmpeg

import (
	"os/exec"
	"syscall"
)

// hideWindow keeps FFmpeg's console out of sight: the app has none of its own.
func hideWindow(cmd *exec.Cmd) {
	const createNoWindow = 0x08000000
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
