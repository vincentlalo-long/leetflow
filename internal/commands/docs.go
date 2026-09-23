package commands

import (
	"fmt"
	"strings"

	"leetcli/internal/config"
)

// CommandDoc holds the human-readable manual entry for one CLI command.
type CommandDoc struct {
	Name        string
	Summary     string
	Usage       string
	Description string
	Examples    []string
}

// CommandDocs maps command names to their manual entries. It lives in the
// commands package so both the TUI (ui package) and the headless CLI can
// render help without an import cycle.
var CommandDocs = map[string]CommandDoc{
	"init": {
		Name:        "init",
		Summary:     "Run interactive first-time setup wizard",
		Usage:       "init [--dir <path>] [--editor <editor>] [--lang <lang>] [--terminal <term>]",
		Description: "Configures machine-specific settings like solution workspace directory, code editor, and terminal layout, saving them safely to config.local.json without touching git.",
		Examples:    []string{"init", "init --dir ~/leetcode --editor nvim"},
	},
	"add": {
		Name:        "add",
		Summary:     "Create a new problem workspace file & README",
		Usage:       "add [problem_number]",
		Description: "Fetches problem details from LeetCode API, prompts for data structure category, creates the problem folder, generates solution template code and README.md description.",
		Examples:    []string{"add 1", "add 15"},
	},
	"add-sol": {
		Name:        "add-sol",
		Summary:     "Add a new solution method to an existing problem",
		Usage:       "add-sol [problem_number]",
		Description: "Allows adding an alternative solution approach (e.g., Solution 2 Two Pointers) with method title, time & space complexities, and source code snippet into the problem file.",
		Examples:    []string{"add-sol 1"},
	},
	"list": {
		Name:        "list",
		Summary:     "List and filter local problems",
		Usage:       "list [--ds <category>] [--difficulty Easy|Medium|Hard] [--unsolved]",
		Description: "Lists all problem files created in your workspace. Supports filtering by data structure category, difficulty, and showing only unsolved problems.",
		Examples:    []string{"list", "list --difficulty Medium", "list --ds array"},
	},
	"search": {
		Name:        "search",
		Summary:     "Search problems by keyword or problem ID",
		Usage:       "search [query]",
		Description: "Searches for problem titles or IDs matching the query string in your local workspace and LeetCode problem library.",
		Examples:    []string{"search two sum", "search 121"},
	},
	"manage-structures": {
		Name:        "manage-structures",
		Summary:     "Add, remove, or list data structure categories",
		Usage:       "manage-structures",
		Description: "Manages data structure category mappings (e.g. array -> array, tree -> tree) used to organize problem folders in your workspace.",
		Examples:    []string{"manage-structures"},
	},
	"stats": {
		Name:        "stats",
		Summary:     "View workspace solving statistics",
		Usage:       "stats",
		Description: "Displays local workspace statistics including total problems created, breakdown by category, and progress tracker summary (solved counts, review queue).",
		Examples:    []string{"stats"},
	},
	"theme": {
		Name:        "theme",
		Summary:     "Change TUI color theme",
		Usage:       "theme",
		Description: "Changes the color palette of the terminal user interface.",
		Examples:    []string{"theme"},
	},
	"daily": {
		Name:        "daily",
		Summary:     "Get and setup LeetCode Daily Challenge",
		Usage:       "daily [--add]",
		Description: "Fetches today's active LeetCode Daily Challenge question, displays title & difficulty, and automatically prompts to create local problem files. --add adds it non-interactively.",
		Examples:    []string{"daily", "daily --add"},
	},
	"random": {
		Name:        "random",
		Summary:     "Get a random LeetCode problem",
		Usage:       "random",
		Description: "Fetches a random problem from LeetCode based on difficulty or category, displaying details and offering to create local files.",
		Examples:    []string{"random"},
	},
	"hint": {
		Name:        "hint",
		Summary:     "View hints for a specific problem",
		Usage:       "hint [problem_number]",
		Description: "Fetches official hints from LeetCode for the specified problem to help you solve it without looking at full solutions.",
		Examples:    []string{"hint 1"},
	},
	"similar": {
		Name:        "similar",
		Summary:     "Find similar problems on LeetCode",
		Usage:       "similar [problem_number]",
		Description: "Displays a list of related / similar LeetCode questions for a given problem number.",
		Examples:    []string{"similar 1"},
	},
	"open": {
		Name:        "open",
		Summary:     "Open problem file in code editor",
		Usage:       "open [problem_number] [--mode auto|split|tmux|wm|simple]",
		Description: "Opens the local problem solution and README in your configured editor (VS Code, Neovim, etc.). Supports split panes (code + README) and Tmux/tiling WM layouts.",
		Examples:    []string{"open 1", "open 1 --mode split", "open 1 --simple"},
	},
	"view": {
		Name:        "view",
		Summary:     "Display problem description with styled markdown & diagrams",
		Usage:       "view [problem_number] [--raw] [--no-pager]",
		Description: "Renders the problem description directly in your terminal using Glamour, with syntax highlighting, borders, and image support (including Kitty graphics protocol).",
		Examples:    []string{"view 1", "view 15", "view 1 --raw"},
	},
	"run": {
		Name:        "run",
		Summary:     "Compile or run code locally",
		Usage:       "run [problem_number]",
		Description: "Compiles and executes your local solution code using installed system compilers/interpreters (g++, gcc, python, go, javac, rustc, node, tsx, dotnet).",
		Examples:    []string{"run 1"},
	},
	"test": {
		Name:        "test",
		Summary:     "Run sample testcases locally or on LeetCode API",
		Usage:       "test [problem_number] [--local]",
		Description: "With --local, builds a harness from the problem's examples (from the local README, falling back to LeetCode) and runs it entirely on your machine without cookies. Without it, sends code to the LeetCode API (requires cookies) and displays Expected vs Actual output.",
		Examples:    []string{"test 1 --local", "test 1"},
	},
	"submit": {
		Name:        "submit",
		Summary:     "Submit solution to LeetCode API",
		Usage:       "submit [problem_number]",
		Description: "Submits your solution code directly to LeetCode API, polls real-time status, and displays Accepted / Wrong Answer / Error details along with Runtime and Memory percentiles (% Beats).",
		Examples:    []string{"submit 1"},
	},
	"sync": {
		Name:        "sync",
		Summary:     "Sync workspace with Git remote repository",
		Usage:       "sync [--progress]",
		Description: "Automates Git workflow: stages all workspace changes, creates a timestamped commit, pulls remote changes with rebase, and pushes to GitHub. --progress also force-stages .leet/progress.json so your solving history & review schedule are backed up.",
		Examples:    []string{"sync", "sync --progress"},
	},
	"profile": {
		Name:        "profile",
		Summary:     "View LeetCode user profile stats",
		Usage:       "profile [username]",
		Description: "Fetches user profile information, global ranking, reputation, and solved question counts from LeetCode GraphQL API.",
		Examples:    []string{"profile vincent"},
	},
	"contest": {
		Name:        "contest",
		Summary:     "View upcoming LeetCode contests",
		Usage:       "contest",
		Description: "Displays upcoming LeetCode Weekly and Biweekly contests along with start times and durations.",
		Examples:    []string{"contest"},
	},
	"config": {
		Name:        "config",
		Summary:     "View and edit CLI settings (base_dir, editor, session, etc.)",
		Usage:       "config [show|open]",
		Description: "Opens an interactive menu to view and modify CLI settings (base workspace directory, default language, text editor, LeetCode cookies, theme) or open config.json directly.",
		Examples:    []string{"config", "config show", "config open", "cfg"},
	},
	"web": {
		Name:        "web",
		Summary:     "Open problem page in default web browser",
		Usage:       "web [problem_number]",
		Description: "Opens the official LeetCode problem page (https://leetcode.com/problems/slug/) directly in your default web browser (Chrome, Edge, Safari...).",
		Examples:    []string{"web 1", "browser two-sum"},
	},
	"timer": {
		Name:        "timer",
		Summary:     "Start practice countdown stopwatch for interview practice",
		Usage:       "timer [minutes]",
		Description: "Starts a background countdown timer (e.g. 20, 30, 45 minutes) to practice solving problems under real interview time pressure.",
		Examples:    []string{"timer 20", "timer 30"},
	},
	"note": {
		Name:        "note",
		Summary:     "Add study notes and key takeaways to problem README",
		Usage:       "note [problem_number]",
		Description: "Appends personal study notes, key tricks, and complexity takeaways into the problem's README.md file.",
		Examples:    []string{"note 1"},
	},
	"clean": {
		Name:        "clean",
		Summary:     "Clean temporary build artifacts (.exe, .class, temp files)",
		Usage:       "clean",
		Description: "Scans workspace recursively and removes temporary build files generated during local execution (leet_run_*, .class, .tmp) to keep repository clean.",
		Examples:    []string{"clean"},
	},
	"readme": {
		Name:        "readme",
		Summary:     "Generate or update workspace root README.md index table",
		Usage:       "readme",
		Description: "Scans workspace problem folders and automatically builds a clickable Markdown index table with problem statistics and relative links to solution files and problem READMEs.",
		Examples:    []string{"readme", "build-readme"},
	},
	"review": {
		Name:        "review",
		Summary:     "Spaced-repetition review queue for solved problems",
		Usage:       "review [number] | --list | --due | --solve <num> | --unsolve <num>",
		Description: "Lists solved problems that are due for revision using a spaced-repetition schedule (1, 3, 7, 15, 30, 60 days). Select a problem to mark it reviewed and schedule the next session. Submit results are recorded automatically in .leet/progress.json.",
		Examples:    []string{"review", "review --due", "review 1", "review --solve 1", "review --list"},
	},
	"clear": {
		Name:        "clear",
		Summary:     "Clear TUI terminal output",
		Usage:       "clear",
		Description: "Clears the scrollback history of the terminal interface.",
		Examples:    []string{"clear"},
	},
	"exit": {
		Name:        "exit",
		Summary:     "Exit the CLI application",
		Usage:       "exit",
		Description: "Saves current configuration and cleanly exits the application.",
		Examples:    []string{"exit", "quit"},
	},
	"doctor": {
		Name:        "doctor",
		Summary:     "Validate workspace config and directory structure",
		Usage:       "doctor",
		Description: "Checks base_dir, config, data structures, cookies, file structure, orphan files, missing READMEs, and suspicious filenames. Reports fixable issues.",
		Examples:    []string{"doctor"},
	},
	"verify": {
		Name:        "verify",
		Summary:     "Auto-test a solution file without interactive prompts",
		Usage:       "verify <file_path> [--local]",
		Description: "Infers problem number from the filename, fetches example test cases (local README or LeetCode API), builds a harness, and runs it. Exits non-zero on failure. Ideal for CI/CD pipelines.",
		Examples:    []string{"verify solutions/array/1_two_sum.cpp --local", "verify main.go"},
	},
	"completion": {
		Name:        "completion",
		Summary:     "Generate shell completion scripts",
		Usage:       "completion [bash|zsh|fish]",
		Description: "Prints tab-completion scripts for bash, zsh, or fish. Add the output to your shell's completion config to enable command and flag completion.",
		Examples:    []string{"completion bash", "completion zsh", "completion fish"},
	},
}

// CommandOrder is the canonical order used when listing commands in help.
var CommandOrder = []string{
	"init", "add", "add-sol", "list", "search", "manage-structures",
	"stats", "theme", "daily", "random", "hint", "similar",
	"view", "open", "web", "run", "test", "submit", "verify", "timer", "note", "review",
	"sync", "clean", "readme", "doctor", "profile", "contest", "config",
	"completion", "clear", "exit",
}

// HelpCommand is the handler for help/man/--help/-h. With no arguments it
// lists all commands; with a command name it prints that command's manual.
func HelpCommand(args []string, cfg *config.Config, ui UI) error {
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		name := strings.TrimSpace(args[0])
		if doc, ok := CommandDocs[name]; ok {
			ui.WriteOutput(MsgPlain, "--- Help: %s ---\n", name)
			ui.WriteString(renderDocPlain(doc))
			return nil
		}
		ui.WriteOutput(MsgError, "No manual found for command '%s'.", name)
		ui.WriteOutput(MsgInfo, "Run 'leet help' to list all commands.")
		return fmt.Errorf("no manual for command '%s'", name)
	}

	ui.WriteOutput(MsgPlain, "leet CLI - Available Commands")
	ui.WriteOutput(MsgPlain, "")
	for _, name := range CommandOrder {
		doc, ok := CommandDocs[name]
		if !ok {
			continue
		}
		ui.WriteOutput(MsgPlain, "  %-18s %s", doc.Name, doc.Summary)
	}
	ui.WriteOutput(MsgPlain, "")
	ui.WriteOutput(MsgInfo, "Use 'leet help <command>' for usage and examples of a specific command.")
	return nil
}

// renderDocPlain renders a manual entry as plain text (headless friendly).
func renderDocPlain(doc CommandDoc) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "  Usage:       leet %s\n", doc.Usage)
	fmt.Fprintf(&sb, "  Summary:     %s\n", doc.Summary)
	fmt.Fprintf(&sb, "  Description: %s\n", doc.Description)
	if len(doc.Examples) > 0 {
		sb.WriteString("  Examples:\n")
		for _, ex := range doc.Examples {
			fmt.Fprintf(&sb, "    leet %s\n", ex)
		}
	}
	return sb.String()
}
