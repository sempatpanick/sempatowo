# Contributing to discordgo-self

Thank you for your interest in contributing to discordgo-self! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please be respectful and considerate in all interactions. We aim to maintain a welcoming and inclusive community.

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/hytams/discordgo-self/issues)
2. If not, create a new issue with:
   - A clear title and description
   - Steps to reproduce the bug
   - Expected vs actual behavior
   - Go version and OS information

### Suggesting Features

1. Check existing issues for similar suggestions
2. Create a new issue with the "enhancement" label
3. Describe the feature and its use case

### Pull Requests

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Run tests: `go test ./...`
5. Run linter: `go vet ./...`
6. Commit with a descriptive message
7. Push to your fork
8. Open a Pull Request

## Development Setup

```bash
# Clone the repository
git clone https://github.com/hytams/discordgo-self.git
cd discordgo-self

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build ./...
```

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add comments for exported functions
- Keep functions focused and small
- Write tests for new features

## Testing

- Write unit tests for new functionality
- Ensure all tests pass before submitting PR
- Include integration tests where applicable

## Questions?

Feel free to open an issue for any questions or concerns!
