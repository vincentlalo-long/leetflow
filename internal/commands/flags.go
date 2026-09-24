package commands

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"

	"leetcli/internal/template"
)

// isTerminalStdin checks if standard input is attached to an interactive terminal.
func isTerminalStdin() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}


// resolveLangFlag maps a --lang value (key, label, or extension) to a language key.
func resolveLangFlag(languages map[string]template.LanguageInfo, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)

	if _, ok := languages[lower]; ok {
		return lower
	}
	if key := template.GetLanguageByExtension(languages, "."+strings.TrimPrefix(lower, ".")); key != "" {
		return key
	}
	for k, info := range languages {
		if strings.EqualFold(info.Label, value) {
			return k
		}
	}
	return ""
}

// parseFlags splits command args into positional args and a --key value / --flag map.
func parseFlags(args []string) (positional []string, flags map[string]string) {
	flags = make(map[string]string)
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "--") {
			key := strings.TrimPrefix(a, "--")
			if eq := strings.IndexByte(key, '='); eq >= 0 {
				flags[key[:eq]] = key[eq+1:]
				continue
			}
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				flags[key] = args[i+1]
				i++
			} else {
				flags[key] = "true"
			}
			continue
		}
		positional = append(positional, a)
	}
	return positional, flags
}

func hasFlag(flags map[string]string, key string) bool {
	v, ok := flags[key]
	return ok && v != "" && v != "false"
}

// HeadlessUI implements commands.UI for non-interactive CLI mode (leet <cmd> [args]).
type HeadlessUI struct{}

func (h HeadlessUI) IsHeadless() bool {
	return true
}

func (h HeadlessUI) PromptText(label string) string {
	if !isTerminalStdin() {
		return ""
	}
	fmt.Printf("%s: ", label)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func (h HeadlessUI) PromptSelect(label string, choices []string) string {
	if !isTerminalStdin() {
		return ""
	}
	if len(choices) == 0 {
		return ""
	}
	fmt.Printf("%s:\n", label)
	for i, choice := range choices {
		fmt.Printf("  [%d] %s\n", i+1, choice)
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("Select option [1-%d] (Enter for 1): ", len(choices))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			return choices[0]
		}
		idx, err := strconv.Atoi(input)
		if err == nil && idx >= 1 && idx <= len(choices) {
			return choices[idx-1]
		}
		fmt.Println("Invalid selection, please try again.")
	}
}

func (h HeadlessUI) PromptConfirm(label string) bool {
	if !isTerminalStdin() {
		return false
	}
	fmt.Printf("%s [y/N]: ", label)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

func (h HeadlessUI) WriteOutput(kind MsgKind, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	switch kind {
	case MsgError:
		fmt.Fprintln(os.Stderr, "✘ "+msg)
	case MsgSuccess:
		fmt.Fprintln(os.Stdout, "✔ "+msg)
	case MsgInfo:
		fmt.Fprintln(os.Stdout, "ℹ "+msg)
	default:
		fmt.Fprintln(os.Stdout, msg)
	}
}

func (h HeadlessUI) WriteString(s string) {
	fmt.Fprint(os.Stdout, s)
}

func (h HeadlessUI) Writef(format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, format, args...)
}
