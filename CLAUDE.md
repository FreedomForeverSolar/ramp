# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Ramp is a CLI tool for managing multi-repository development workflows using git worktrees and automated setup scripts. It helps developers work on features across multiple repositories simultaneously by creating isolated working directories.

## Core Commands

### Build and Development
- `go build -o ramp .` - Build the project
- `./install.sh` - Build and install to /usr/local/bin (requires sudo)
- `go run . --help` - Run without building to see available commands
- `go test ./...` - Run tests (standard Go testing)

### CLI Usage
- `ramp init` - Initialize a project by cloning all configured repositories
- `ramp new <feature-name>` - Create feature branches with git worktrees for all repos
- `ramp --help` - Show help information

## Architecture

### Command Structure
The application uses the Cobra CLI framework with commands organized in `cmd/`:
- `cmd/root.go` - Main command definition and CLI entry point
- `cmd/init.go` - Repository initialization logic  
- `cmd/new.go` - Feature branch and worktree creation

### Core Packages
- `internal/config/` - Configuration file parsing (.ramp/ramp.yaml)
- `internal/git/` - Git operations (clone, worktree management)

### Configuration
Projects require a `.ramp/ramp.yaml` file with:
```yaml
name: project-name
repos:
  - path: git@github.com:owner/repo.git
    default_branch: main
setup: scripts/setup.sh  # optional
```

### Directory Structure
- `source/` - Original repository clones
- `trees/<feature-name>/` - Git worktrees for feature development
- `.ramp/ramp.yaml` - Project configuration

### Key Functions
- `config.FindRampProject()` - Searches up directory tree for .ramp/ramp.yaml
- `config.LoadConfig()` - Parses YAML configuration 
- `git.CreateWorktree()` - Creates git worktrees with feature branches
- `runSetupScript()` - Executes post-creation setup scripts with environment variables

The tool enables multi-repo feature development by creating git worktrees that allow working on the same feature across multiple repositories simultaneously, with optional automated setup scripts for environment configuration.