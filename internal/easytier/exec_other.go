//go:build !windows

package easytier

import "os/exec"

func hideWindow(cmd *exec.Cmd) {}
