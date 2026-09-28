package cli

import (
	"fmt"
	"os"

	"leetcli/internal/commands"
	"leetcli/internal/config"
)

// Run executes the lightweight, non-interactive CLI and returns an exit code.
func Run(args []string) int {
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
	}
	if cfg == nil {
		cfg = config.Default()
		cfg.ResolveBaseDir()
	}

	if len(args) == 0 {
		_ = commands.Registry["help"](nil, cfg, commands.HeadlessUI{})
		return 0
	}

	cmdName := args[0]
	cmdArgs := args[1:]
	handler, ok := commands.Registry[cmdName]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown command: %q. Run 'leet help' for available commands.\n", cmdName)
		return 1
	}

	if wantsHelp(cmdArgs) && cmdName != "help" && cmdName != "man" {
		if err := commands.Registry["help"]([]string{cmdName}, cfg, commands.HeadlessUI{}); err != nil {
			return 1
		}
		return 0
	}

	if cfg.NeedsSetup() && !isMetaCommand(cmdName) {
		fmt.Fprintln(os.Stderr, "Workspace is not configured. Run 'leet init' first.")
		return 1
	}

	if err := handler(cmdArgs, cfg, commands.HeadlessUI{}); err != nil {
		return 1
	}
	return 0
}

func wantsHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func isMetaCommand(cmd string) bool {
	switch cmd {
	case "init", "setup", "help", "man", "--help", "-h", "version", "--version", "-v", "completion", "doctor":
		return true
	default:
		return false
	}
}
