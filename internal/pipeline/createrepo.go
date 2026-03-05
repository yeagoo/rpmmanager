package pipeline

import (
	"context"
	"fmt"
	"os/exec"
)

// RunCreaterepo runs createrepo_c on the given directory.
func RunCreaterepo(ctx context.Context, createrepoPath, repoDir string, logWriter *LogWriter) error {
	logWriter.WriteLog("Running createrepo_c on %s", repoDir)

	cmd := exec.CommandContext(ctx, createrepoPath,
		"--general-compress-type=xz",
		"--update",
		repoDir,
	)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("createrepo_c failed on %s: %w", repoDir, err)
	}
	return nil
}
