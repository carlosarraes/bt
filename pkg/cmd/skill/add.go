package skill

import (
	"context"
	"fmt"
)

type AddCmd struct {
	Force bool
}

func (cmd *AddCmd) Run(ctx context.Context) error {
	if isInstalled() {
		version, _ := installedVersion()

		linked, skipped, err := createSymlinks(cmd.Force)
		if err != nil {
			return fmt.Errorf("failed to create symlinks: %w", err)
		}

		if len(linked) > 0 {
			fmt.Printf("bt skill v%s already installed\n", version)
			fmt.Println("Linked to:")
			for _, name := range linked {
				fmt.Printf("  - %s\n", name)
			}
		} else if skipped > 0 {
			fmt.Printf("bt skill v%s installed, all agents linked\n", version)
		}
		return nil
	}

	fmt.Println("Fetching bt skill from GitHub...")
	version, err := fetchAndStore()
	if err != nil {
		return fmt.Errorf("failed to fetch skill: %w", err)
	}

	linked, skipped, err := createSymlinks(cmd.Force)
	if err != nil {
		return fmt.Errorf("failed to create symlinks: %w", err)
	}

	fmt.Printf("Installed bt skill v%s\n", version)
	if len(linked) > 0 {
		fmt.Println("Linked to:")
		for _, name := range linked {
			fmt.Printf("  - %s\n", name)
		}
	} else if skipped > 0 {
		fmt.Println("All agents already linked")
	} else {
		fmt.Println("No AI agents detected. Install Claude, Cursor, or Codex and run 'bt skill add' again.")
	}

	return nil
}
