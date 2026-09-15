package commands

import (
	"fmt"

	"leetcli/internal/config"
)

func ManageStructures(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Manage Data Structures ---\n")

	action := ui.PromptSelect("What would you like to do?", []string{
		"List all data structures",
		"Add new data structure",
		"Remove data structure",
	})
	if action == "" {
		return nil
	}

	switch action {
	case "List all data structures":
		listStructures(cfg, ui)
	case "Add new data structure":
		if err := addNewStructure(cfg, ui); err != nil {
			return err
		}
	case "Remove data structure":
		if err := removeStructure(cfg, ui); err != nil {
			return err
		}
	}
	return nil
}

func listStructures(cfg *config.Config, ui UI) {
	structures := cfg.GetDataStructures()
	if len(structures) == 0 {
		ui.WriteOutput(MsgInfo, "No data structures found!")
		return
	}
	ui.WriteOutput(MsgPlain, "Available Data Structures:")
	for name, folder := range structures {
		ui.WriteOutput(MsgPlain, "  %-20s %s", name, folder)
	}
}

func addNewStructure(cfg *config.Config, ui UI) error {
	name := ui.PromptText("Data structure name (e.g., tree)")
	if name == "" {
		ui.WriteOutput(MsgError, "Data structure name cannot be empty!")
		return fmt.Errorf("data structure name cannot be empty")
	}

	folder := ui.PromptText("Folder name (press Enter for same)")
	if folder == "" {
		folder = name
	}

	if cfg.AddDataStructure(name, folder) {
		if err := cfg.Save(); err != nil {
			ui.WriteOutput(MsgError, "Failed to save config: %v", err)
			return fmt.Errorf("failed to save config: %w", err)
		}
		ui.WriteOutput(MsgSuccess, "Added data structure: %s -> %s", name, folder)
		return nil
	}
	ui.WriteOutput(MsgError, "Data structure '%s' already exists!", name)
	return nil
}

func removeStructure(cfg *config.Config, ui UI) error {
	structures := cfg.GetDataStructures()
	if len(structures) == 0 {
		ui.WriteOutput(MsgInfo, "No data structures to remove!")
		return nil
	}

	names := make([]string, 0, len(structures))
	for n := range structures {
		names = append(names, n)
	}

	name := ui.PromptSelect("Select data structure to remove", names)
	if name == "" {
		return nil
	}

	confirm := ui.PromptConfirm("Remove '" + name + "'?")
	if !confirm {
		ui.WriteOutput(MsgInfo, "Operation cancelled")
		return nil
	}

	if cfg.RemoveDataStructure(name) {
		if err := cfg.Save(); err != nil {
			ui.WriteOutput(MsgError, "Failed to save config: %v", err)
			return fmt.Errorf("failed to save config: %w", err)
		}
		ui.WriteOutput(MsgSuccess, "Removed data structure: %s", name)
		return nil
	}
	ui.WriteOutput(MsgError, "Data structure '%s' not found!", name)
	return nil
}
