package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/algebananazzzzz/bytecanteen/internal/auth"
	"github.com/algebananazzzzz/bytecanteen/internal/clock"
	"github.com/algebananazzzzz/bytecanteen/internal/config"
	"github.com/algebananazzzzz/bytecanteen/internal/run"
	"github.com/algebananazzzzz/bytecanteen/internal/schedule"
	"github.com/algebananazzzzz/bytecanteen/internal/tui"
)

// Stamped by GoReleaser at build time via -ldflags -X.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Load endpoint config from .env (./.env or ~/.config/canteen/.env) before any
	// subcommand reads it; real environment variables still take precedence.
	config.LoadDotenv()

	root := &cobra.Command{
		Use:     "canteen",
		Short:   "Auto-book canteen lunch",
		Version: fmt.Sprintf("%s (commit %s, built %s)", version, commit, date),
		Run: func(cmd *cobra.Command, args []string) {
			if err := tui.Run(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}

	bookCmd := &cobra.Command{
		Use:   "book",
		Short: "Run a booking (headless)",
		RunE: func(cmd *cobra.Command, args []string) error {
			dry, _ := cmd.Flags().GetBool("dry")
			d, err := run.LoadDeps()
			if err != nil {
				return err
			}
			res, err := run.Book(d, dry)
			if err != nil {
				return err
			}
			fmt.Println(res.Booked)
			return nil
		},
	}
	bookCmd.Flags().Bool("dry", false, "force dry-run")
	root.AddCommand(bookCmd)

	root.AddCommand(&cobra.Command{
		Use:   "menu",
		Short: "Print upcoming menus and update the dish catalog",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := run.LoadDeps()
			if err != nil {
				return err
			}
			return run.Menu(d)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "auth",
		Short: "One-time browser QR login",
		RunE: func(cmd *cobra.Command, args []string) error {
			return auth.Login()
		},
	})

	schedCmd := &cobra.Command{Use: "schedule", Short: "Manage the weekly job"}
	schedCmd.AddCommand(&cobra.Command{
		Use: "on", RunE: func(cmd *cobra.Command, args []string) error {
			bin, _ := os.Executable()
			d, err := run.LoadDeps()
			if err != nil {
				return err
			}
			loc, err := time.LoadLocation(d.Cfg.Schedule.TZ)
			if err != nil {
				return fmt.Errorf("bad timezone %q: %w", d.Cfg.Schedule.TZ, err)
			}
			wdMap := map[string]time.Weekday{"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6}
			openTime := clock.NextOpen(time.Now(), wdMap[d.Cfg.Schedule.Weekday], d.Cfg.Schedule.Hour, d.Cfg.Schedule.Minute, loc)
			l := openTime.Local()
			return schedule.On(bin, int(l.Weekday()), l.Hour(), l.Minute())
		},
	})
	schedCmd.AddCommand(&cobra.Command{
		Use: "off", RunE: func(cmd *cobra.Command, args []string) error { return schedule.Off() },
	})
	root.AddCommand(schedCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
