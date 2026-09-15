package commands

import (
	"fmt"
	"strings"

	"leetcli/internal/config"
)

// CompletionCommand is the handler for `leet completion`. It prints shell
// completion scripts for bash, zsh, or fish.
func CompletionCommand(args []string, cfg *config.Config, ui UI) error {
	shell := ""
	if len(args) > 0 {
		shell = strings.ToLower(strings.TrimSpace(args[0]))
	}
	if shell == "" {
		shell = "bash"
	}

	commands := make([]string, 0, len(CommandOrder))
	for _, name := range CommandOrder {
		if _, ok := CommandDocs[name]; ok {
			commands = append(commands, name)
		}
	}

	switch shell {
	case "bash":
		ui.WriteString(bashCompletionScript(commands))
	case "zsh":
		ui.WriteString(zshCompletionScript(commands))
	case "fish":
		ui.WriteString(fishCompletionScript(commands))
	default:
		ui.WriteOutput(MsgError, "Unsupported shell '%s'. Use: bash, zsh, fish", shell)
		return fmt.Errorf("unsupported shell '%s'", shell)
	}

	ui.WriteOutput(MsgInfo, "Add the output to your shell's completion config:")
	ui.WriteOutput(MsgInfo, "  bash: leet completion bash >> ~/.bashrc")
	ui.WriteOutput(MsgInfo, "  zsh:  leet completion zsh > ~/.zfunc/_leet")
	ui.WriteOutput(MsgInfo, "  fish: leet completion fish > ~/.config/fish/completions/leet.fish")
	return nil
}

func bashCompletionScript(commands []string) string {
	var sb strings.Builder
	sb.WriteString("#!/usr/bin/env bash\n")
	sb.WriteString("_leet_completions() {\n")
	sb.WriteString("    local cur=\"${COMP_WORDS[COMP_CWORD]}\"\n")
	sb.WriteString("    local prev=\"${COMP_WORDS[COMP_CWORD-1]}\"\n")
	sb.WriteString("    local commands='")
	for i, c := range commands {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(c)
	}
	sb.WriteString("'\n")
	sb.WriteString("\n")
	sb.WriteString("    # Sub-commands for specific parents.\n")
	sb.WriteString("    case \"$prev\" in\n")
	sb.WriteString("        help|man)\n")
	sb.WriteString("            COMPREPLY=( $(compgen -W \"$commands\" -- \"$cur\") )\n")
	sb.WriteString("            return 0\n")
	sb.WriteString("            ;;\n")
	sb.WriteString("        test)\n")
	sb.WriteString("            COMPREPLY=( $(compgen -W \"--local\" -- \"$cur\") )\n")
	sb.WriteString("            return 0\n")
	sb.WriteString("            ;;\n")
	sb.WriteString("        timer)\n")
	sb.WriteString("            COMPREPLY=( $(compgen -W \"15 20 30 45 60\" -- \"$cur\") )\n")
	sb.WriteString("            return 0\n")
	sb.WriteString("            ;;\n")
	sb.WriteString("        config|cfg)\n")
	sb.WriteString("            COMPREPLY=( $(compgen -W \"show open\" -- \"$cur\") )\n")
	sb.WriteString("            return 0\n")
	sb.WriteString("            ;;\n")
	sb.WriteString("        completion)\n")
	sb.WriteString("            COMPREPLY=( $(compgen -W \"bash zsh fish\" -- \"$cur\") )\n")
	sb.WriteString("            return 0\n")
	sb.WriteString("            ;;\n")
	sb.WriteString("    esac\n")
	sb.WriteString("\n")
	sb.WriteString("    COMPREPLY=( $(compgen -W \"$commands\" -- \"$cur\") )\n")
	sb.WriteString("}\n")
	sb.WriteString("complete -F _leet_completions leet\n")
	return sb.String()
}

func zshCompletionScript(commands []string) string {
	var sb strings.Builder
	sb.WriteString("#compdef leet\n\n")
	sb.WriteString("# Zsh completion for leet CLI\n")
	sb.WriteString("_leet() {\n")
	sb.WriteString("    local -a commands\n")
	sb.WriteString("    commands=(\n")
	for _, c := range commands {
		doc, ok := CommandDocs[c]
		summary := ""
		if ok {
			summary = doc.Summary
		}
		fmt.Fprintf(&sb, "        '%s:%s'\n", c, summary)
	}
	sb.WriteString("    )\n\n")
	sb.WriteString("    _arguments -C \\\n")
	sb.WriteString("        '1:command:->command' \\\n")
	sb.WriteString("        '*::arg:->args'\n\n")
	sb.WriteString("    case $state in\n")
	sb.WriteString("        command)\n")
	sb.WriteString("            _describe 'leet commands' commands\n")
	sb.WriteString("            ;;\n")
	sb.WriteString("        args)\n")
	sb.WriteString("            case $words[1] in\n")
	sb.WriteString("                test)\n")
	sb.WriteString("                    _arguments '--local[run test cases locally without cookies]'\n")
	sb.WriteString("                    ;;\n")
	sb.WriteString("                timer)\n")
	sb.WriteString("                    _values 'minutes' 15 20 30 45 60\n")
	sb.WriteString("                    ;;\n")
	sb.WriteString("                config|cfg)\n")
	sb.WriteString("                    _values 'option' show open\n")
	sb.WriteString("                    ;;\n")
	sb.WriteString("                completion)\n")
	sb.WriteString("                    _values 'shell' bash zsh fish\n")
	sb.WriteString("                    ;;\n")
	sb.WriteString("            esac\n")
	sb.WriteString("            ;;\n")
	sb.WriteString("    esac\n")
	sb.WriteString("}\n\n")
	sb.WriteString("_leet \"$@\"\n")
	return sb.String()
}

func fishCompletionScript(commands []string) string {
	var sb strings.Builder
	sb.WriteString("# Fish completion for leet CLI\n\n")
	for _, c := range commands {
		doc, ok := CommandDocs[c]
		summary := ""
		if ok {
			summary = doc.Summary
		}
		fmt.Fprintf(&sb, "complete -c leet -f -n '__fish_use_subcommand' -a '%s' -d '%s'\n", c, summary)
	}
	sb.WriteString("\n# Sub-command specific completions\n")
	sb.WriteString("complete -c leet -f -n '__fish_seen_subcommand_from test' -l local -d 'Run locally without cookies'\n")
	sb.WriteString("complete -c leet -f -n '__fish_seen_subcommand_from timer' -a '15 20 30 45 60'\n")
	sb.WriteString("complete -c leet -f -n '__fish_seen_subcommand_from config cfg' -a 'show open'\n")
	sb.WriteString("complete -c leet -f -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish'\n")
	return sb.String()
}
