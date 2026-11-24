//go:build !windows

package ffmpeg

import "os/exec"

// hideWindow does nothing where a process opens no window of its own.
func hideWindow(cmd *exec.Cmd) {}
