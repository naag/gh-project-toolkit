# GitHub Project Toolkit

A tool for managing GitHub projects (v2) that helps synchronize fields between project boards.

## Features

- Sync fields between GitHub project boards
- Support for both organization and user-scoped projects
- Automatic issue detection across projects
- Flexible field mapping configuration

## Installation

### Using go install

```bash
go install github.com/naag/gh-project-toolkit/cmd/gh-project-toolkit@latest
```

### Using released binaries

Download the latest binary for your platform from the [GitHub Releases page](https://github.com/naag/gh-project-toolkit/releases).

### Building from source

```bash
# Clone the repository
git clone https://github.com/naag/gh-project-toolkit.git
cd gh-project-toolkit

# Build the binary
make build

# Binary will be available in bin/gh-project-toolkit
```

## Usage

### Syncing Fields Between Projects

The tool can automatically sync field values between two GitHub projects for all issues that exist in both projects:

```bash
gh-project-toolkit sync-fields \
  --source "https://github.com/orgs/myorg/projects/123" \
  --target "https://github.com/orgs/myorg/projects/456" \
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
  --source "https://github.com/orgs/myorg/projects/123" \
  --target "https://github.com/orgs/myorg/projects/456" \
  --field-mapping "Start date=Start" \
  --field-mapping "End date=End" \
  --issue "https://github.com/org/repo/issues/1" \
  --issue "https://github.com/org/repo/issues/2"
```

### Authentication

The tool requires a GitHub personal access token with appropriate permissions:
- Set the `GITHUB_TOKEN` environment variable with your token
- Token needs `project` scope for reading/writing project data
- For organization projects, the token needs access to the organization

### Options

- `--source`: Source project URL (e.g., https://github.com/orgs/org/projects/123)
- `--target`: Target project URL (e.g., https://github.com/users/user/projects/456)
- `--field-mapping`: Field mapping in the format 'source=target' (can be specified multiple times)
- `--auto-detect-issues`: Automatically detect and sync all issues present in both projects
- `--issue`: GitHub issue URL (can be specified multiple times, not needed with --auto-detect-issues)
- `-v, --verbose`: Enable verbose logging (use -vv for HTTP traffic)

## Development

### Requirements

- Go 1.21 or later
- Make
- [golangci-lint](https://golangci-lint.run/) (optional, for linting)

### Commands

- `make build`: Build the binary (output to bin/gh-project-toolkit)
- `make test`: Run tests
- `make lint`: Run linters
- `make clean`: Clean build artifacts
- `make install-tools`: Install development tools (like golangci-lint)

### Release Process

Releases are automatically created when pushing a new tag:

```bash
# Create and push a new version tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

This will trigger the release workflow which:
1. Runs tests and linting
2. Creates a GitHub release
3. Builds binaries for multiple platforms
4. Attaches binaries to the release

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request 