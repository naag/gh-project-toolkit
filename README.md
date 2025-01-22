# GitHub Project Toolkit

A tool for managing GitHub projects (v2).

## Features

- Sync fields between GitHub project boards

## Installation

```bash
go install github.com/naag/gh-project-toolkit/cmd/gh-project-toolkit@latest
```

## Usage

### Syncing Fields Between Projects

The tool can automatically sync field values between two GitHub projects for all issues that exist in both projects:

```bash
gh-project-toolkit sync-fields \
  --org myorg \
  --source-project 123 \
  --target-project 456 \
  --field-mapping "Start date=Start" \
  --field-mapping "End date=End" \
  --auto-detect-issues
```

This will:
1. Find all issues that exist in both projects
2. For each common issue, copy the field values from source to target project using the provided mappings

You can also specify individual issues manually if needed:

```bash
gh-project-toolkit sync-fields \
  --org myorg \
  --source-project 123 \
  --target-project 456 \
  --field-mapping "Start date=Start" \
  --field-mapping "End date=End" \
  --issue "https://github.com/org/repo/issues/1" \
  --issue "https://github.com/org/repo/issues/2"
```

### Authentication

The tool requires a GitHub personal access token with appropriate permissions:
- Set the `GITHUB_TOKEN` environment variable with your token
- Token needs `project` scope for reading/writing project data

### Options

- `--org`: GitHub organization name (mutually exclusive with --user)
- `--user`: GitHub username for user-scoped projects (mutually exclusive with --org)
- `--source-project`: Source project number
- `--target-project`: Target project number
- `--field-mapping`: Field mapping in the format 'source=target' (can be specified multiple times)
- `--auto-detect-issues`: Automatically detect and sync all issues present in both projects
- `--issue`: GitHub issue URL (can be specified multiple times, not needed with --auto-detect-issues)
- `-v, --verbose`: Enable verbose logging of HTTP traffic

## Development

### Requirements

- Go 1.21 or later

### Commands

- `make build`: Build the binary
- `make test`: Run tests
- `make lint`: Run linters
- `make clean`: Clean build artifacts 