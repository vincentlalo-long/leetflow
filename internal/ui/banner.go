package ui

import "fmt"

const banner = `
  ██╗  ██████╗ ███████╗ █████╗ ██╗
  ██║  ██╔══██╗██╔════╝██╔══██╗██║
  ██║  ██║  ██║█████╗  ███████║██║
  ██║  ██║  ██║██╔══╝  ██╔══██║██║
  ██║  ██████╔╝███████╗██║  ██║███████╗
  ╚═╝  ╚═════╝ ╚══════╝╚═╝  ╚═╝╚══════╝
`

func RenderBanner() string {
	return BannerStyle.Render(banner) + "\n"
}

const helpText = `Available commands:
  add              Create a new problem
  add-sol          Add a solution to a problem
  list             List and filter problems (--ds, --difficulty, --unsolved)
  search           Search problems by name or number
  manage-structures Manage data structures
  stats            Show problem statistics + progress
  theme            Change the UI theme
  daily            Get and add LeetCode daily challenge
  random           Get a random LeetCode problem
  hint             Get hints for a specific problem
  similar          Find similar problems
  open             Open problem description and code editor
  run              Compile or run code locally
  test             Run LeetCode testcases for a solution
  submit           Submit solution to LeetCode and show benchmark
  review           Spaced-repetition review queue for solved problems
  sync             Sync LeetCode workspace with Git
  profile          View LeetCode user profile stats
  contest          View upcoming LeetCode contests
  help, /help      Show this help message
  exit, quit       Exit the CLI

Tip: any command also works from the shell:  leet list --difficulty Easy
`

func RenderHelp() string {
	return helpText
}

func RenderTips() string {
	return fmt.Sprintf(
		"%sTips for getting started:%s\n"+
			"1. Ask questions, edit files, or run commands.\n"+
			"2. Be specific for the best results.\n"+
			"3. %s/help%s for more information.\n",
		White, "",
		CommandStyle.Render(""), White,
	)
}
