//go:build !windows

package pty

// GetChildProcesses returns stub child process names on non-Windows platforms.
func GetChildProcesses(parentPid int) ([]string, error) {
	return nil, nil
}
