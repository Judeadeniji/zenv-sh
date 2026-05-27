//go:build windows

package commands

import (
	"os"
	"os/exec"
)

// execCommand runs the target binary as a child process and waits for it to exit.
//
// On Windows, syscall.Exec (execve) does not exist, so we fork and wait instead.
// Unlike the Unix path, this leaves the parent process alive with decrypted env
// vars in memory until cmd.Run() returns. Callers should already have zeroed key
// material (dek, hmacKey) before calling this; here we also drop the env slice
// references after the child exits so the GC can collect sooner.
//
// Note: Go strings are immutable — we cannot zero individual secret values — but
// clearing the slice at least removes the references.
func execCommand(binary string, args []string, env []string) error {
	cmd := exec.Command(binary, args[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	// Best-effort: drop env entry references so the GC can reclaim sooner.
	for i := range env {
		env[i] = ""
	}

	return err
}
