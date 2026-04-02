# Contributing to mcp-proxy

Thank you for your interest in contributing to mcp-proxy! This document provides guidelines and information for contributors.

## How to Contribute

### Reporting Issues

- Use [GitHub Issues](https://github.com/agent-receipts/mcp-proxy/issues) to report bugs or suggest features.
- Check existing issues before creating a new one to avoid duplicates.
- Provide as much detail as possible, including steps to reproduce bugs.

### Submitting Changes

1. **Fork the repository** and create a feature branch from `main`.
2. **Make your changes** with clear, descriptive commits.
3. **Add or update tests** as appropriate.
4. **Ensure all tests pass** by running `go test ./...`.
5. **Open a pull request** against `main`.

### Pull Request Process

- Keep pull requests focused on a single change.
- Provide a clear description of what the PR does and why.
- Link any related issues.
- PRs require at least one maintainer review before merging.
- Address review feedback promptly; we aim for constructive and collaborative reviews.

## Development Setup

```bash
git clone https://github.com/agent-receipts/mcp-proxy.git
cd mcp-proxy
go mod download
go build ./...
go test ./...
```

## Community Guidelines

We are committed to providing a welcoming and inclusive experience for everyone. All participants are expected to treat others with respect and professionalism. Harassment or exclusionary behavior is not tolerated.

## License

By contributing to this project, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE). All new files must include the appropriate license header where applicable.
