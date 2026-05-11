package runner

import (
	"io"
	"os/exec"
)

// Run executes `make target` in dir, streaming stdout and stderr to out.
// It returns the exit error if the command fails.
func Run(dir, target string, out io.Writer) error {
	cmd := exec.Command("make", target)
	cmd.Dir = dir
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}
