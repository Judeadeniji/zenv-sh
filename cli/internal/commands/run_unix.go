//go:build !windows

package commands

import "syscall"

// execCommand replaces the current process with the target binary via execve(2).
// No Go runtime cleanup occurs after this point — callers must zero all key
// material before calling this function.
func execCommand(binary string, args []string, env []string) error {
	return syscall.Exec(binary, args, env)
}
