package commands

import (
	"fmt"
	"os"
	"strings"

	"leetcli/internal/template"
)

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
	fmt.Fprintf(os.Stderr, "✘ Cannot prompt in non-interactive mode: %s\n", label)
	return ""
}

func (h HeadlessUI) PromptSelect(label string, choices []string) string {
	fmt.Fprintf(os.Stderr, "✘ Cannot prompt in non-interactive mode: %s\n", label)
	return ""
}

func (h HeadlessUI) PromptConfirm(label string) bool {
	fmt.Fprintf(os.Stderr, "✘ Cannot prompt in non-interactive mode: %s\n", label)
	return false
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
