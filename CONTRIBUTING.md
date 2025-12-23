# Contributing to Cimon

Thank you for your interest in contributing to cimon! This document provides guidelines and information for contributing to the project.

## Code of Conduct

By participating in this project, you agree to maintain a respectful, inclusive environment.

## How to Contribute

### Reporting Issues

We appreciate bug reports, feature requests, and other issues!

- Search existing issues first to avoid duplicates
- Use clear, descriptive titles
- Include steps to reproduce the issue
- Provide relevant environment details:
  - OS and version (e.g., `uname -a` or `go version`)
  - Cimon version: `cimon --version`
  - Repository and branch you're monitoring

### Pull Requests

We welcome pull requests for bug fixes, new features, and documentation improvements!

#### Development Workflow

1. Fork the repository
2. Create a new branch for your feature or bugfix
3. Make your changes
4. Run tests: `make test`
5. Ensure code follows existing patterns and style
6. Commit your changes with clear messages
7. Push to your fork and submit a pull request

#### Code Style

- Follow existing code patterns and naming conventions
- Add comments to explain complex logic
- Run `make lint` before committing
- Ensure all tests pass: `make test`
- Avoid introducing new linter warnings

#### Testing

- Write tests for new functionality
- Aim for good test coverage (aim for 70%+)
- Test both happy paths and error cases
- Run `make test` before committing

#### Commit Messages

- Use clear, concise commit messages
- Format: `type: brief description`
  - `feat:` for new features
  - `fix:` for bug fixes
  - `docs:` for documentation changes
  - `test:` for test additions/changes
- Reference relevant issues in commit body

### Feature Suggestions

If you have ideas for new features or improvements:

- Check the [ROADMAP.md](ROADMAP.md) for planned features
- Search existing issues to avoid duplications
- Open a new issue with the `enhancement` label

### Documentation

- Update [README.md](README.md) if changing user-facing behavior
- Update [CHANGELOG.md](CHANGELOG.md) for notable changes
- Ensure examples remain accurate after API changes

## Development Setup

### Prerequisites

- Go 1.23 or later
- `gh` CLI tool (for GitHub authentication during development)
- Make (usually available, or check `Makefile`)

### Building

```bash
# Build the binary
make build

# Run tests
make test

# Run linter
make lint
```

### Project Structure

```
cmd/cimon/          # CLI entry point
internal/
  config/         # Configuration parsing and validation
  gh/              # GitHub API client and models
  git/             # Git repository detection and parsing
  notify/          # Desktop notifications and hooks
  tui/             # Terminal UI (Bubble Tea)
```

## Getting Help

If you need help or have questions:

- Check existing [issues](https://github.com/lance0/cimon/issues) first
- Ask in [GitHub Discussions](https://github.com/lance0/cimon/discussions)
- Read the [README](README.md) and [ROADMAP](ROADMAP.md)

Thank you for contributing to cimon! 🚀
