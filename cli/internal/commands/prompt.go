package commands

import (
	"bufio"
	"bytes"
	"fmt"
	"os"

	"golang.org/x/term"
)

// PromptSecret reads a sensitive value from the user with the given label
// (e.g. "vault key", "API token").
//
// TTY: input is masked via terminal raw mode (no echo).
// Pipe: reads from stdin directly, strips the trailing newline.
//
// Returns a []byte so the caller can zero it after use — do not convert to string.
func PromptSecret(label string) ([]byte, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return promptMasked(label)
	}
	return promptPiped(label)
}

func promptMasked(label string) ([]byte, error) {
	fmt.Fprintf(os.Stderr, "Enter %s: ", label)
	key, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr) // newline after the hidden input
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", label, err)
	}
	if len(key) == 0 {
		return nil, fmt.Errorf("%s must not be empty", label)
	}
	return key, nil
}

func promptPiped(label string) ([]byte, error) {
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading %s from stdin: %w", label, err)
		}
		return nil, fmt.Errorf("%s must not be empty", label)
	}
	key := bytes.TrimRight(scanner.Bytes(), "\r\n")
	if len(key) == 0 {
		return nil, fmt.Errorf("%s must not be empty", label)
	}
	// scanner.Bytes() is a shared buffer — copy before the scanner is GC'd.
	out := make([]byte, len(key))
	copy(out, key)
	return out, nil
}
