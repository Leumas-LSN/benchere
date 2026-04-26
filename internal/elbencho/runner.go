package elbencho

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type RunConfig struct {
	Hosts       []string
	ConfigFile  string
	Targets     []string // e.g. ["/dev/sdb", "/dev/sdc"]
	LiveCSVPath string
	CSVPath     string
	Label       string
}

func Run(ctx context.Context, cfg RunConfig) error {
	args := []string{
		"--hosts", strings.Join(cfg.Hosts, ","),
		"--configfile", cfg.ConfigFile,
		"--livecsv", cfg.LiveCSVPath,
		"--csvfile", cfg.CSVPath,
		"--label", cfg.Label,
	}
	args = append(args, cfg.Targets...)

	cmd := exec.CommandContext(ctx, "elbencho", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("elbencho run %s: %w", cfg.Label, err)
	}
	return nil
}
