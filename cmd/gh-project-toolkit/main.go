package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/naag/gh-project-toolkit/internal/github/client"
	"github.com/naag/gh-project-toolkit/internal/tools/sync_fields"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:          "gh-project-toolkit",
	Short:        "GitHub Project Toolkit - Tools for managing GitHub projects",
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var level slog.Level
		switch verboseLevel {
		case 0:
			level = slog.LevelInfo
		case 1, 2:
			level = slog.LevelDebug
		}
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
		slog.SetDefault(logger)
	},
}

var syncFieldsCmd = &cobra.Command{
	Use:          "sync-fields",
	Short:        "Sync fields between GitHub project boards",
	SilenceUsage: true,
	RunE:         runSyncFields,
}

var (
	sourceProjectURL string
	targetProjectURL string
	issues           []string
	fieldMappings    []string
	verboseLevel     int
	autoDetectIssues bool
	dryRun           bool
)

func init() {
	rootCmd.AddCommand(syncFieldsCmd)

	syncFieldsCmd.Flags().StringVar(&sourceProjectURL, "source", "", "Source project URL (e.g., https://github.com/orgs/org/projects/123)")
	syncFieldsCmd.Flags().StringVar(&targetProjectURL, "target", "", "Target project URL (e.g., https://github.com/users/user/projects/456)")
	syncFieldsCmd.Flags().StringArrayVar(&issues, "issue", nil, "GitHub issue URL (can be specified multiple times)")
	syncFieldsCmd.Flags().StringArrayVar(&fieldMappings, "field-mapping", nil, "Field mapping in the format 'source=target' (can be specified multiple times)")
	syncFieldsCmd.Flags().CountVarP(&verboseLevel, "verbose", "v", "Verbosity level (-v for debug logs, -vv for debug logs and HTTP traffic)")
	syncFieldsCmd.Flags().BoolVar(&autoDetectIssues, "auto-detect-issues", false, "Automatically detect and sync all issues present in both projects")
	syncFieldsCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Run in dry run mode (no mutations will be performed)")

	// Mark required flags
	requiredFlags := []string{"source", "target", "field-mapping"}
	for _, flag := range requiredFlags {
		if err := syncFieldsCmd.MarkFlagRequired(flag); err != nil {
			panic(fmt.Sprintf("failed to mark flag %s as required: %v", flag, err))
		}
	}
}

func runSyncFields(cmd *cobra.Command, args []string) error {
	client, err := client.NewGraphQLClient(verboseLevel >= 2)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	service := sync_fields.NewService(client, dryRun)

	if len(issues) == 0 && !autoDetectIssues {
		return fmt.Errorf("no issues specified and --auto-detect-issues not enabled")
	}

	if err := service.SyncFields(context.Background(), sourceProjectURL, targetProjectURL, issues, fieldMappings); err != nil {
		return fmt.Errorf("failed to sync fields: %w", err)
	}

	if dryRun {
		slog.Info("dry run completed successfully")
	} else {
		slog.Info("sync completed successfully")
	}
	return nil
}
