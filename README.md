# GitHub Project Toolkit

A toolkit for managing GitHub projects and issues.

## Features

- Sync fields between GitHub project boards

## Installation

```bash
make build
```

## Usage

### Sync Fields

Sync fields between two GitHub project boards:

```bash
gh-project-toolkit sync-fields \
  --org myorg \
  --source-project 824 \
  --target-project 825 \
  --issue https://github.com/org/repo/issues/1 \
  --issue https://github.com/org/repo/issues/2 \
  --field-mapping 'start=Start date' \
  --field-mapping 'end=End date'
```

## Development

### Requirements

- Go 1.21 or later

### Commands

- `make build`: Build the binary
- `make test`: Run tests
- `make lint`: Run linters
- `make clean`: Clean build artifacts 