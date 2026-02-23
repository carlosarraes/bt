package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type RemoveCmd struct{}

func (cmd *RemoveCmd) Run(ctx context.Context) error {
	if !isInstalled() {
		return fmt.Errorf("skill is not installed")
	}

	removed, err := removeSymlinks()
	if err != nil {
		return fmt.Errorf("failed to remove symlinks: %w", err)
	}

	dir, err := skillDir()
	if err != nil {
		return err
	}

	os.RemoveAll(filepath.Join(dir, "bt"))
	os.Remove(filepath.Join(dir, ".version"))
	os.Remove(filepath.Join(dir, ".last_check"))

	fmt.Println("Removed bt skill")
	if len(removed) > 0 {
		fmt.Println("Unlinked from:")
		for _, name := range removed {
			fmt.Printf("  - %s\n", name)
		}
	}
	return nil
}
