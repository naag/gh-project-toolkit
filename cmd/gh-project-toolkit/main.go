package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/naag/gh-project-toolkit/internal/github"
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
}

var syncFieldsCmd = &cobra.Command{
	Use:          "sync-fields",
	Short:        "Sync fields between GitHub project boards",
	SilenceUsage: true,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Validate that exactly one of --org or --user is specified
		if (org == "") == (user == "") {
			return fmt.Errorf("exactly one of --org or --user must be specified")
		}
		return nil
	},
	RunE: runSyncFields,
}

var (
	org           string
	user          string
	sourceProject int
	targetProject int
	issues        []string
	fieldMappings []string
	verbose       bool
)

func init() {
	rootCmd.AddCommand(syncFieldsCmd)

	syncFieldsCmd.Flags().StringVar(&org, "org", "", "GitHub organization name (mutually exclusive with --user)")
	syncFieldsCmd.Flags().StringVar(&user, "user", "", "GitHub username for user-scoped projects (mutually exclusive with --org)")
	syncFieldsCmd.Flags().IntVar(&sourceProject, "source-project", 0, "Source project number")
	syncFieldsCmd.Flags().IntVar(&targetProject, "target-project", 0, "Target project number")
	syncFieldsCmd.Flags().StringArrayVar(&issues, "issue", nil, "GitHub issue URL (can be specified multiple times)")
	syncFieldsCmd.Flags().StringArrayVar(&fieldMappings, "field-mapping", nil, "Field mapping in the format 'source=target' (can be specified multiple times)")
	syncFieldsCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging of HTTP traffic")

	for _, flag := range []string{"source-project", "target-project", "issue", "field-mapping"} {
		if err := syncFieldsCmd.MarkFlagRequired(flag); err != nil {
			// This should never happen as we're using predefined flags
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
	client, err := github.NewGraphQLClient(verbose)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// Create sync service
	service := sync.NewService(client)

	// Determine owner type and login
	var ownerType github.OwnerType
	var ownerLogin string
	if user != "" {
		ownerType = github.OwnerTypeUser
		ownerLogin = user
	} else {
		ownerType = github.OwnerTypeOrg
		ownerLogin = org
	}

	if err := service.SyncFields(context.Background(), ownerType, ownerLogin, sourceProject, targetProject, issues, mappings); err != nil {
		return fmt.Errorf("failed to sync fields: %w", err)
	}

	slog.Info("sync completed successfully")
	return nil
}
