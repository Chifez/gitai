package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "gitai",
	Short: "AI-powered git commit CLI",
	Long:  "gitai reads staged git changes, generates a well-structured commit message using an LLM, presents it for review, then commits and optionally pushes.",
}

// Execute runs the root command. Called from main.go.
func Execute() {
	// Create context cancelled on SIGINT/SIGTERM
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("gitai v{{.Version}}\n")
}
