package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/naag/gh-project-toolkit/internal/github"
	"github.com/naag/gh-project-toolkit/internal/github/projecturl"
	"github.com/naag/gh-project-toolkit/internal/sync"
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
		// Configure logging based on verbose level
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
)

func init() {
	rootCmd.AddCommand(syncFieldsCmd)

	syncFieldsCmd.Flags().StringVar(&sourceProjectURL, "source", "", "Source project URL (e.g., https://github.com/orgs/org/projects/123)")
	syncFieldsCmd.Flags().StringVar(&targetProjectURL, "target", "", "Target project URL (e.g., https://github.com/users/user/projects/456)")
	syncFieldsCmd.Flags().StringArrayVar(&issues, "issue", nil, "GitHub issue URL (can be specified multiple times)")
	syncFieldsCmd.Flags().StringArrayVar(&fieldMappings, "field-mapping", nil, "Field mapping in the format 'source=target' (can be specified multiple times)")
	syncFieldsCmd.Flags().CountVarP(&verboseLevel, "verbose", "v", "Verbosity level (-v for debug logs, -vv for debug logs and HTTP traffic)")
	syncFieldsCmd.Flags().BoolVar(&autoDetectIssues, "auto-detect-issues", false, "Automatically detect and sync all issues present in both projects")

	// Mark required flags
	requiredFlags := []string{"source", "target", "field-mapping"}
	for _, flag := range requiredFlags {
		if err := syncFieldsCmd.MarkFlagRequired(flag); err != nil {
			panic(fmt.Sprintf("failed to mark flag %s as required: %v", flag, err))
		}
	}
}

func runSyncFields(cmd *cobra.Command, args []string) error {
	// Parse field mappings
	mappings := make([]sync.FieldMapping, 0, len(fieldMappings))
	for _, mapping := range fieldMappings {
		parts := strings.Split(mapping, "=")
		if len(parts) != 2 {
			return fmt.Errorf("invalid field mapping format: %s", mapping)
		}
		mappings = append(mappings, sync.FieldMapping{
			SourceField: strings.TrimSpace(parts[0]),
			TargetField: strings.TrimSpace(parts[1]),
		})
	}

	// Initialize GitHub client
	client, err := github.NewGraphQLClient(verboseLevel >= 2)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// Create sync service
	service := sync.NewService(client)

	// Parse project URLs
	sourceInfo, err := projecturl.Parse(sourceProjectURL)
	if err != nil {
		return fmt.Errorf("invalid source project URL: %w", err)
	}

	targetInfo, err := projecturl.Parse(targetProjectURL)
	if err != nil {
		return fmt.Errorf("invalid target project URL: %w", err)
	}

	// If no issues are specified and auto-detect is not enabled, return an error
	if len(issues) == 0 && !autoDetectIssues {
		return fmt.Errorf("no issues specified and --auto-detect-issues not enabled")
	}

	// Call SyncFields with empty issues slice if auto-detect is enabled
	if err := service.SyncFields(context.Background(), sourceInfo.OwnerType, sourceInfo.OwnerLogin, sourceInfo.ProjectNumber, targetInfo.ProjectNumber, issues, mappings); err != nil {
		return fmt.Errorf("failed to sync fields: %w", err)
	}

	slog.Info("sync completed successfully")
	return nil
}
