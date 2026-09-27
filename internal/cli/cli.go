package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

var commands = [...]string{"plan", "install", "setup", "verify"}

// Commands returns the four installer conveniences accepted by the host binary.
func Commands() []string {
	return append([]string(nil), commands[:]...)
}

// Run executes only the Task 1 command boundary and returns a process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		usage(stdout)
		return 0
	}
	if len(args) == 0 || !recognized(args[0]) {
		return usageError(stderr)
	}

	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}
	err := flags.Parse(args[1:])
	if errors.Is(err, flag.ErrHelp) && len(args) == 2 && (args[1] == "-h" || args[1] == "--help") {
		usage(stdout)
		return 0
	}
	if err != nil || flags.NArg() != 0 || len(args) != 1 {
		return usageError(stderr)
	}

	fmt.Fprintln(stderr, "command not implemented")
	return 1
}

func recognized(command string) bool {
	for _, candidate := range commands {
		if command == candidate {
			return true
		}
	}
	return false
}

func usage(output io.Writer) {
	fmt.Fprintln(output, "usage: hermes-repokit <plan|install|setup|verify> [--help]")
}

func usageError(output io.Writer) int {
	fmt.Fprintln(output, "usage error")
	usage(output)
	return 2
}
