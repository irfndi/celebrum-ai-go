# Contributing to Celebrum AI

Thank you for your interest in contributing to Celebrum AI! This document provides guidelines and instructions for contributing to this project.

## 📋 Table of Contents

- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Contributing Process](#contributing-process)
- [Code Standards](#code-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Commit Message Guidelines](#commit-message-guidelines)
- [Pull Request Process](#pull-request-process)
- [Issue Reporting](#issue-reporting)

## 🚀 Getting Started

### Prerequisites

- **Go 1.22+** - [Installation Guide](https://golang.org/doc/install)
- **Bun 1.0+** - [Installation Guide](https://bun.sh/docs/installation)
- **Docker & Docker Compose** - [Installation Guide](https://docs.docker.com/get-docker/)
- **PostgreSQL 15+** (or use Docker)
- **Redis 7+** (or use Docker)

### Development Setup

1. **Fork and Clone**
   ```bash
   git clone https://github.com/irfndi/celebrum-ai-go.git
   cd celebrum-ai-go
   ```

2. **Environment Setup**
   ```bash
   # Copy environment template
   cp .env.example .env
   
   # Edit .env with your configurations
   # Required: TELEGRAM_BOT_TOKEN, DATABASE_URL, REDIS_URL
   ```

3. **Start Development Environment**
   ```bash
   # Using Docker Compose (Recommended)
   make docker-run
   
   # Or run locally
   make setup
   make dev
   ```

4. **Verify Setup**
   ```bash
   make test
   make health-check
   ```

## 🔄 Contributing Process

### 1. Choose an Issue
- Check [Issues](https://github.com/irfndi/celebrum-ai-go/issues) for open tasks
- Look for issues labeled `good first issue` or `help wanted`
- Create a new issue if you want to suggest a feature or report a bug

### 2. Create a Branch
```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-description
```

### 3. Make Changes
- Follow our [Code Standards](#code-standards)
- Write tests for new functionality
- Update documentation as needed

### 4. Test Your Changes
```bash
# Run all tests
make test

# Run specific test suites
make test-go          # Go tests
make test-ccxt        # CCXT service tests
make test-integration  # Integration tests
```

### 5. Submit Pull Request
- Ensure all tests pass
- Follow our [Pull Request Process](#pull-request-process)

## 🎯 Code Standards

### Go Code Style
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Use `golangci-lint` for linting
- Write clear, concise function and variable names
- Include comprehensive godoc comments

### TypeScript/JavaScript Style
- Follow [TypeScript Style Guide](https://www.typescriptlang.org/docs/handbook/declaration-files/do-s-and-don-ts.html)
- Use ESLint configuration provided
- Prefer async/await over callbacks
- Use meaningful type definitions

### API Design
- RESTful API principles
- Consistent response formats
- Proper HTTP status codes
- Comprehensive error handling
- Rate limiting considerations

## 🧪 Testing

### Test Categories
- **Unit Tests**: Test individual functions/methods
- **Integration Tests**: Test component interactions
- **End-to-End Tests**: Test complete user workflows
- **Performance Tests**: Test system performance under load

### Writing Tests
- **Go**: Use standard `testing` package
- **TypeScript**: Use Jest for CCXT service
- **Mock external dependencies** when possible
- **Test both success and error cases**

### Running Tests
```bash
# All tests
make test

# Specific components
make test-go
make test-ccxt
make test-docker

# With coverage
make test-coverage
```

## 📚 Documentation

### Code Documentation
- **Go**: Use godoc format
- **TypeScript**: Use JSDoc format
- **README files**: Keep README.md updated
- **API Documentation**: Update OpenAPI specs

### Documentation Standards
- Clear, concise explanations
- Include code examples
- Update documentation with code changes
- Use consistent formatting

## 📝 Commit Message Guidelines

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types
- **feat**: New feature
- **fix**: Bug fix
- **docs**: Documentation changes
- **style**: Code style changes (formatting, etc.)
- **refactor**: Code refactoring
- **test**: Adding or updating tests
- **chore**: Maintenance tasks

### Examples
```
feat(api): add funding rate endpoint
fix(ccxt): handle connection timeouts
docs: update deployment instructions
test: add integration tests for arbitrage detection
```

## 🔄 Pull Request Process

### Before Submitting
1. **Sync with main branch**
   ```bash
   git fetch origin
   git rebase origin/main
   ```

2. **Run all tests**
   ```bash
   make test
   make lint
   ```

3. **Update documentation** if needed

### PR Requirements
- **Clear title and description**
- **Reference related issues** (use `Fixes #123` or `Closes #123`)
- **Include tests** for new functionality
- **Update documentation** for user-facing changes
- **Pass all CI checks**

### PR Template
```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Tests added/updated
```

## 🐛 Issue Reporting

### Bug Reports
Use the bug report template and include:
- **Clear description** of the issue
- **Steps to reproduce**
- **Expected vs actual behavior**
- **Environment details** (OS, Go version, etc.)
- **Logs or error messages**

### Feature Requests
Use the feature request template and include:
- **Clear description** of the proposed feature
- **Use cases and benefits**
- **Possible implementation approach**
- **Impact on existing functionality**

## 🛠️ Development Commands

### Makefile Targets
```bash
make help              # Show all available commands
make setup             # Initial setup
make dev               # Start development environment
make test              # Run all tests
make test-coverage     # Run tests with coverage
make lint              # Run linting
make build             # Build all components
make docker-run        # Run with Docker Compose
make clean             # Clean build artifacts
```

### Docker Commands
```bash
# Development
make docker-run

# Production
make docker-build
make docker-push

# Testing
make test-docker
```

## 🆘 Getting Help

### Resources
- **Documentation**: Check `/docs` directory
- **Issues**: Search existing issues
- **Discussions**: Use GitHub Discussions for questions

### Contact
- **Issues**: Create an issue for bugs or feature requests
- **Discussions**: Use GitHub Discussions for general questions
- **Email**: [Contact maintainers](mailto:irfandi@example.com)

## 📄 License

By contributing to Celebrum AI, you agree that your contributions will be licensed under the same license as the project (MIT License).

---

Thank you for contributing to Celebrum AI! 🚀