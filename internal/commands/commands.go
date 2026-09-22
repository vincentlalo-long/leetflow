package commands

import (
	"leetcli/internal/config"
)

type MsgKind int

const (
	MsgPlain MsgKind = iota
	MsgInfo
	MsgSuccess
	MsgError
	MsgOutput
	MsgClear
)

type UI interface {
	PromptText(label string) string
	PromptSelect(label string, choices []string) string
	PromptConfirm(label string) bool
	WriteOutput(kind MsgKind, format string, args ...interface{})
	WriteString(s string)
	Writef(format string, args ...interface{})
}

// Handler runs one CLI command. It returns a non-nil error when the command
// failed so headless (script/CI) invocations can exit non-zero. The TUI ignores
// the return value (errors are already reported via ui.WriteOutput).
type Handler func(args []string, cfg *config.Config, ui UI) error

var Registry = map[string]Handler{
	"init":              InitCommand,
	"setup":             InitCommand,
	"add":               AddProblem,
	"add-sol":           AddSolution,
	"list":              ListProblems,
	"search":            SearchProblems,
	"manage-structures": ManageStructures,
	"stats":             Stats,
	"theme":             Theme,
	"daily":             Daily,
	"random":            Random,
	"hint":              Hint,
	"similar":           Similar,
	"open":              OpenProblem,
	"run":               RunProblem,
	"test":              TestProblem,
	"submit":            SubmitProblem,
	"sync":              Sync,
	"profile":           Profile,
	"contest":           Contest,
	"config":            ManageConfig,
	"cfg":               ManageConfig,
	"web":               OpenWebProblem,
	"browser":           OpenWebProblem,
	"timer":             Timer,
	"note":              AddNote,
	"clean":             CleanWorkspace,
	"readme":            BuildReadme,
	"build-readme":      BuildReadme,
	"review":            ReviewQueue,
	"doctor":            DoctorCommand,
	"verify":            VerifyCommand,
	"completion":        CompletionCommand,
	"version":           VersionCommand,
	"--version":         VersionCommand,
	"-v":                VersionCommand,
	"help":              HelpCommand,
	"man":               HelpCommand,
	"--help":            HelpCommand,
	"-h":                HelpCommand,
}
