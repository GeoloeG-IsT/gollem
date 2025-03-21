# Contributing to Gollem

Thank you for your interest in contributing to Gollem! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md) to foster an open and welcoming environment.

## How to Contribute

There are many ways to contribute to Gollem:

1. Reporting bugs
2. Suggesting enhancements
3. Writing documentation
4. Submitting code changes
5. Reviewing pull requests

### Reporting Bugs

If you find a bug, please create an issue using the bug report template. Include as much detail as possible:

- A clear and descriptive title
- Steps to reproduce the issue
- Expected behavior
- Actual behavior
- Environment details (OS, Go version, etc.)
- Any relevant logs or error messages

### Suggesting Enhancements

If you have an idea for an enhancement, please create an issue using the feature request template. Include:

- A clear and descriptive title
- A detailed description of the proposed enhancement
- Any relevant examples or use cases
- If applicable, potential implementation approaches

### Writing Documentation

Documentation improvements are always welcome. You can:

- Fix typos or clarify existing documentation
- Add examples or tutorials
- Document undocumented features
- Improve API documentation

### Submitting Code Changes

1. Fork the repository
2. Create a new branch for your changes
3. Make your changes
4. Write or update tests for your changes
5. Ensure all tests pass
6. Submit a pull request

#### Development Setup

1. Clone your fork:
   ```bash
   git clone https://github.com/yourusername/gollem.git
   cd gollem
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Run tests:
   ```bash
   go test ./...
   ```

#### Pull Request Process

1. Update the README.md or documentation with details of changes if applicable
2. Update the CHANGELOG.md with details of changes
3. The PR should work with Go 1.18 and later
4. The PR will be merged once it receives approval from maintainers

### Code Style

- Follow standard Go code style and conventions
- Use `gofmt` to format your code
- Follow the [Effective Go](https://golang.org/doc/effective_go) guidelines
- Run `golangci-lint` before submitting your code

## Adding New Providers

To add a new LLM provider:

1. Create a new package in `pkg/providers/`
2. Implement the `core.LLMProvider` interface
3. Add tests for the new provider
4. Update documentation to include the new provider
5. Add an example demonstrating the new provider

## License

By contributing to Gollem, you agree that your contributions will be licensed under the project's [MIT License](LICENSE).
