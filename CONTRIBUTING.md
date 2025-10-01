# Contributing to Ramp

Thank you for your interest in contributing to Ramp! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)

## Code of Conduct

We expect all contributors to be respectful and constructive in their interactions with the community. Please be kind and courteous in discussions, issues, and pull requests.

## Getting Started

### Reporting Bugs

If you find a bug, please create an issue on GitHub with:

- A clear, descriptive title
- Steps to reproduce the issue
- Expected behavior
- Actual behavior
- Your environment (OS, Go version, Ramp version)
- Any relevant logs or error messages

### Suggesting Features

Feature suggestions are welcome! Please create an issue with:

- A clear description of the feature
- The problem it solves
- Example use cases
- Any potential implementation ideas

## Development Setup

### Prerequisites

- Go 1.24.5 or later
- Git
- Basic familiarity with git worktrees (helpful but not required)

### Setting Up Your Development Environment

1. **Fork the repository** on GitHub

2. **Clone your fork**:
   ```bash
   git clone git@github.com:YOUR-USERNAME/ramp.git
   cd ramp
   ```

3. **Add upstream remote**:
   ```bash
   git remote add upstream git@github.com:FreedomForeverSolar/ramp.git
   ```

4. **Install dependencies**:
   ```bash
   go mod download
   ```

5. **Build the project**:
   ```bash
   go build -o ramp .
   ```

6. **Run tests**:
   ```bash
   go test ./...
   ```

### Running Ramp Locally

You can run your local development version without installing:

```bash
go run . --help
```

Or build and install to test the full installation:

```bash
./install.sh
```

## How to Contribute

### Finding Issues to Work On

- Check the [issue tracker](https://github.com/FreedomForeverSolar/ramp/issues) for open issues
- Issues labeled `good first issue` are great for new contributors
- Issues labeled `help wanted` are ready to be worked on

### Creating a Branch

Create a feature branch from `main`:

```bash
git checkout main
git pull upstream main
git checkout -b feature/your-feature-name
```

Use descriptive branch names:
- `feature/add-interactive-mode` for new features
- `fix/worktree-creation-error` for bug fixes
- `docs/update-readme` for documentation changes

## Coding Standards

### General Guidelines

- **Write small, composable functions**: Keep functions focused and under 40 lines of code when possible
- **Use descriptive names**: Variable and function names should clearly indicate their purpose
- **Add comments**: Explain complex logic and the "why" behind decisions
- **Follow Go conventions**: Use `gofmt` and `golint` to ensure code style consistency

### Go-Specific Standards

- Run `go fmt` before committing
- Use meaningful error messages
- Avoid global variables
- Keep packages focused on a single responsibility
- Document exported functions with comments

### Project Structure

The project follows this structure:

```
cmd/           - CLI command definitions (using Cobra)
internal/      - Internal packages (config, git, ports, ui)
  config/      - Configuration file parsing
  git/         - Git operations and worktree management
  ports/       - Port allocation management
  ui/          - User interface and progress feedback
```

When adding new functionality:
- Add new commands to `cmd/`
- Add supporting logic to appropriate `internal/` packages
- Keep business logic out of CLI command handlers

## Testing

### Running Tests

Run all tests:
```bash
go test ./...
```

Run tests with verbose output:
```bash
go test -v ./...
```

Run tests for a specific package:
```bash
go test ./internal/config
```

### Writing Tests

- Add tests for new functionality
- Add tests to reproduce bugs before fixing them
- Use table-driven tests for multiple similar test cases
- Test both success and error cases
- Mock external dependencies (git commands, file system operations)

Example test structure:

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   "test",
            want:    "expected",
            wantErr: false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("FunctionName() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("FunctionName() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Pull Request Process

### Before Submitting

1. **Update your branch** with the latest changes from upstream:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run tests** and ensure they pass:
   ```bash
   go test ./...
   ```

3. **Run formatting**:
   ```bash
   go fmt ./...
   ```

4. **Build the project** to ensure it compiles:
   ```bash
   go build -o ramp .
   ```

### Submitting Your Pull Request

1. Push your branch to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

2. Create a pull request on GitHub with:
   - A clear title describing the change
   - A description that includes:
     - What the PR does
     - Why the change is needed
     - How to test the changes
     - Any breaking changes or migration notes
   - Link to any related issues using `Fixes #123` or `Closes #123`

3. Wait for review and address any feedback

### Pull Request Guidelines

- Keep PRs focused on a single feature or fix
- Break large changes into smaller, reviewable PRs when possible
- Update documentation if you're changing functionality
- Add or update tests for your changes
- Ensure CI checks pass
- Respond to review feedback promptly

### After Your PR is Merged

- Delete your feature branch
- Pull the latest changes from upstream:
  ```bash
  git checkout main
  git pull upstream main
  ```

## Questions?

If you have questions about contributing, feel free to:

- Open an issue with your question
- Check existing issues and discussions for similar questions
- Reach out to maintainers

Thank you for contributing to Ramp! 🚀
