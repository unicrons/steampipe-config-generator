package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version, Commit, and Date are set via -ldflags at build time by GoReleaser.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// NewVersionCmd builds the "version" subcommand.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "steampipe-config-generator %s (commit %s, built %s)\n", Version, Commit, Date)
			return nil
		},
	}
}
