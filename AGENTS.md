# AI Assistant Guidelines for ghtkn

This document contains common guidelines for AI assistants working on the ghtkn project.
Individual AI-specific documents (like CLAUDE.md, CLINE.md) should reference this guide.

## Language

This project uses **English** for all code comments, documentation, and communication.

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/) specification:

### Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Common Types

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process or auxiliary tools
- `ci`: Changes to CI configuration files and scripts

### Examples

```
feat: add GitHub token management via keyring
fix: handle empty configuration file correctly
docs: add function documentation to controller package
chore(deps): update dependency aquaproj/aqua-registry to v4.403.0
```

## Code Validation

After making code changes, **always run** the following commands to validate and test:

### Validation (go vet)

```bash
cmdx v
```
This command runs `go vet ./...` to check for common Go mistakes.

### Testing

```bash
cmdx t
```
This command runs all tests in the project.

Both commands should pass before committing changes.

## Project Structure

```
ghtkn/
├── cmd/            # Main applications
├── pkg/            # Go packages
│   ├── cli/        # CLI interface layer
│   ├── config/     # Configuration management
│   └── controller/ # Utility functions
├── testdata/       # Test fixtures
├── json-schema/    # JSON schema definitions
└── scripts/        # Build and utility scripts
```

## Package Responsibilities

### pkg/cli
Command-line interface layer that handles command parsing, flag processing, and routing to appropriate subcommands.

### pkg/config
Configuration management including reading, parsing, and validating .ghtkn.yaml files.

### pkg/controller
Business logic layer containing:

## Testing

### Test Framework Guidelines

- **DO NOT** use `testify` for writing tests
- **DO** use `google/go-cmp` for comparing expected and actual values
- Use standard Go testing package (`testing`) for all tests

### Running Tests

- Run all tests: `cmdx t` or `go test ./...`
- Run specific package tests: `go test ./pkg/controller/initcmd`
- Generate coverage: `./scripts/coverage.sh`

## Dependencies

This project uses:

- [aqua](https://aquaproj.github.io/) for tool version management
- [cmdx](https://github.com/suzuki-shunsuke/cmdx) for task runner
- [goreleaser](https://goreleaser.com/) for releases

## Code Style Guidelines

1. Follow standard Go conventions
2. Use meaningful variable and function names
3. Add comments for exported functions and types
4. Keep functions focused and small
5. Handle errors explicitly
6. Use context for cancellation and timeouts
7. Always end files with a newline character

## Pull Request Process

1. Create a feature branch from `main`
2. Make changes and ensure `cmdx v` and `cmdx t` pass
3. Write clear commit messages following Conventional Commits
4. Create PR with descriptive title and body
5. Wait for CI checks to pass
6. Request review if needed

## Important Commands

```bash
# Validate code (go vet)
cmdx v

# Run tests
cmdx t

# Build the project
go build ./cmd/ghtkn

# Generate JSON schema
cmdx js

# Run ghtkn locally
go run ./cmd/ghtkn run
```

## GitHub Actions Integration

The project includes GitHub Actions for:

- Testing on multiple platforms
- Linting and validation
- Release automation
- Security scanning

## Configuration

## Environment Variables

## Debugging

Enable debug logging:

```bash
export GHTKN_LOG_LEVEL=debug
```

## Common Tasks

### Adding a New Command

1. Create new package under `pkg/cli/`
2. Implement command structure with `urfave/cli/v3`
3. Add controller logic under `pkg/controller/`
4. Register command in `pkg/cli/runner.go`
5. Add tests for new functionality

## File Naming Conventions

- Go source files: lowercase with underscores (e.g., `parse_line.go`)
- Test files: append `_test.go` to the source file name
- Internal test files: append `_internal_test.go` for internal testing

## Error Handling

- Always check and handle errors explicitly
- Use `fmt.Errorf` with `%w` for wrapping errors
- Add context to errors to aid debugging
- Use structured logging with slog

## Documentation

- Add package-level documentation comments
- Document all exported functions, types, and constants
- Use examples in documentation where helpful
- Keep README and other docs up to date

## Resources

- [Project README](README.md)
- [Contributing Guidelines](CONTRIBUTING.md)
- [Installation Guide](INSTALL.md)
- [Usage Documentation](USAGE.md)
