package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/skalluru/velocix/internal/config"
	"github.com/skalluru/velocix/internal/manager"
	"github.com/skalluru/velocix/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start the terminal UI dashboard",
	RunE:  runTUI,
}

var tuiPollSecs int

func init() {
	tuiCmd.Flags().IntVar(&tuiPollSecs, "poll-interval", 0, "Poll interval in seconds (default: from config or 30)")
}

func runTUI(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	if tuiPollSecs != 0 {
		cfg.PollInterval = time.Duration(tuiPollSecs) * time.Second
	}

	// Log to file when TUI is active (stdout is owned by Bubble Tea)
	logFile, err := os.OpenFile(
		fmt.Sprintf("%s/velocix.log", cfg.DataDir),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644,
	)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer logFile.Close()
	logger := slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr := manager.New(ctx, cfg, logger)
	defer mgr.Stop()

	active := cfg.GetActiveOrg()
	org := ""
	if active != nil {
		org = active.Organization
	}

	if err := tui.Run(mgr.GetActiveStore(), org, logger); err != nil {
		return err
	}

	cancel()
	return nil
}
