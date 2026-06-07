package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/algebananazzzzz/nybble/internal/auth"
	"github.com/algebananazzzzz/nybble/internal/config"
	"github.com/algebananazzzzz/nybble/internal/run"
	"github.com/algebananazzzzz/nybble/internal/tui"
)

// Stamped by GoReleaser at build time via -ldflags -X.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Load endpoint config from .env (./.env or ~/.config/nybble/.env) before any
	// subcommand reads it; real environment variables still take precedence.
	config.LoadDotenv()

	root := &cobra.Command{
		Use:     "nybble",
		Short:   "Automatically book your canteen lunch from your preferences",
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

	clearCmd := &cobra.Command{
		Use:     "clear",
		Aliases: []string{"logout"},
		Short:   "Clear all local data (clean slate; keeps .env endpoints)",
		RunE: func(cmd *cobra.Command, args []string) error {
			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				fmt.Print("This erases all local data (cookies, config, favorites, catalog) but keeps .env. Continue? [y/N]: ")
				var resp string
				fmt.Scanln(&resp)
				if resp != "y" && resp != "Y" {
					fmt.Println("cancelled")
					return nil
				}
			}
			if err := auth.Clear(); err != nil {
				return err
			}
			fmt.Println("✓ Clean slate — run `nybble auth` to set up again.")
			return nil
		},
	}
	clearCmd.Flags().Bool("yes", false, "skip the confirmation prompt")
	root.AddCommand(clearCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
