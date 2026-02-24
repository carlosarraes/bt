package pick

import (
	"context"
	"fmt"
	"os"

	"github.com/carlosarraes/bt/pkg/git"
)

type ContinueCmd struct {
	NoColor bool
}

func (cmd *ContinueCmd) Run(ctx context.Context) error {
	repoDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	return git.CherryPickContinue(repoDir)
}
