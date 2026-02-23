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
		return fmt.Errorf("skill already installed. Use 'bt skill update' to update or 'bt skill remove' first")
	}

	fmt.Println("Fetching bt skill from GitHub...")
	version, err := fetchAndStore()
	if err != nil {
		return fmt.Errorf("failed to fetch skill: %w", err)
	}

	linked, err := createSymlinks(cmd.Force)
	if err != nil {
		return fmt.Errorf("failed to create symlinks: %w", err)
	}

	fmt.Printf("Installed bt skill v%s\n", version)
	if len(linked) > 0 {
		fmt.Println("Linked to:")
		for _, name := range linked {
			fmt.Printf("  - %s\n", name)
		}
	} else {
		fmt.Println("No AI agents detected. Install Claude, Cursor, or Codex and run 'bt skill add' again.")
	}

	return nil
}
