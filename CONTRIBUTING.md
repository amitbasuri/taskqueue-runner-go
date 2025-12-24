# Contributing to TaskQueue-Go

First off, thank you for considering contributing to TaskQueue-Go! It's people like you that make this project better for everyone.

## 🎯 How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When you create a bug report, include as many details as possible:

**Bug Report Template:**
```markdown
**Describe the bug**
A clear description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Start with '...'
2. Execute '...'
3. See error

**Expected behavior**
What you expected to happen.

**Environment:**
- OS: [e.g., Ubuntu 22.04]
- Go version: [e.g., 1.23]
- PostgreSQL version: [e.g., 16.1]

**Additional context**
Logs, screenshots, or any other relevant information.
```

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, include:

- **Use a clear and descriptive title**
- **Provide a detailed description** of the suggested enhancement
- **Explain why this enhancement would be useful** to most users
- **List some examples** of how it would be used

### Pull Requests

1. **Fork the repo** and create your branch from `main`
2. **Add tests** if you've added code that should be tested
3. **Ensure the test suite passes** (`make test-integration`)
4. **Make sure your code lints** (golangci-lint will run in CI)
5. **Write a clear commit message** following conventional commits

**Commit Message Format:**
```
type(scope): subject

body (optional)

footer (optional)
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting, missing semicolons, etc.
- `refactor`: Code change that neither fixes a bug nor adds a feature
- `test`: Adding missing tests
- `chore`: Changes to build process or auxiliary tools

**Examples:**
```
feat(worker): add support for task cancellation

- Implement context cancellation for running tasks
- Add CancelTask API endpoint
- Update tests to cover cancellation scenarios

Closes #123
```

```
fix(storage): prevent duplicate task execution on lock expiration

Workers now re-validate lock ownership before marking task complete.

Fixes #456
```

## 🏗️ Development Setup

### Prerequisites

- Go 1.22 or higher
- PostgreSQL 16+
- Docker & Docker Compose
- make

### Local Development

1. **Clone your fork:**
```bash
git clone https://github.com/YOUR_USERNAME/taskqueue-go.git
cd taskqueue-go
```

2. **Install dependencies:**
```bash
make deps
```

3. **Start local environment:**
```bash
make docker-up
```

4. **Run tests:**
```bash
make test-integration
```

5. **Access dashboard:**
```bash
open http://localhost:8080
```

### Project Structure

```
taskqueue-go/
├── cmd/
│   ├── server/          # API server entry point
│   └── worker/          # Worker entry point
├── internal/
│   ├── api/             # HTTP handlers
│   ├── config/          # Configuration management
│   ├── models/          # Domain models
│   ├── storage/         # Storage interfaces and implementations
│   └── worker/          # Worker logic and task handlers
├── db/migrations/       # Database migrations
├── tests/               # Integration tests
├── k8s/                 # Kubernetes manifests
└── web/                 # Dashboard UI
```

### Adding a New Task Handler

1. Create handler in `internal/worker/handlers/`:
```go
package handlers

import (
    "context"
    "encoding/json"
)

type MyTaskHandler struct{}

func (h *MyTaskHandler) Type() string {
    return "my_task"
}

func (h *MyTaskHandler) Execute(ctx context.Context, payload json.RawMessage) error {
    // Your logic here
    return nil
}
```

2. Register in `internal/worker/registry.go`:
```go
registry.Register(&handlers.MyTaskHandler{})
```

3. Add tests in `tests/integration.test.js`:
```javascript
test('should process my_task successfully', async (t) => {
    // Test implementation
});
```

## 🧪 Testing Guidelines

### Unit Tests
```bash
go test ./...
```

### Integration Tests
```bash
make test-integration      # Docker Compose
make test-integration-k8s  # Kubernetes
```

### Test Coverage
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**Coverage Requirements:**
- New features should maintain >70% coverage
- Bug fixes must include regression tests

## 📝 Code Style

We use `golangci-lint` for linting. The configuration is in `.golangci.yml`.

**Run linter locally:**
```bash
golangci-lint run
```

**Key style guidelines:**
- Use `gofmt` for formatting
- Write clear, self-documenting code
- Add comments for exported functions and complex logic
- Follow [Effective Go](https://go.dev/doc/effective_go)

## 📄 Documentation

- Update README.md if you change functionality
- Add godoc comments for exported types and functions
- Update API documentation for new endpoints
- Include examples for complex features

## ❓ Questions?

Feel free to ask questions by:
- Opening a discussion on GitHub
- Commenting on related issues
- Reaching out to maintainers

## 📜 Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

---

Thank you for contributing! 🎉
