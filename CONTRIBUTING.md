# Contributing to Columbina

Thank you for your interest in contributing to Columbina! Whether you are reporting bugs, proposing new features, improving documentation, or submitting code changes, your contributions are welcome.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Enhancements](#suggesting-enhancements)
  - [Submitting Pull Requests](#submitting-pull-requests)
- [Local Development Setup](#local-development-setup)
  - [Prerequisites](#prerequisites)
  - [Setting Up the Project](#setting-up-the-project)
  - [Environment Configuration](#environment-configuration)
  - [Running the Bot](#running-the-bot)
- [Development Guidelines](#development-guidelines)
  - [Code Style & Formatting](#code-style--formatting)
  - [Persona & Prompt Integrity](#persona--prompt-integrity)
  - [Branch Naming & Commit Conventions](#branch-naming--commit-conventions)
- [Pull Request Checklist](#pull-request-checklist)

---

## Code of Conduct

Please maintain a respectful, welcoming, and collaborative environment. Treat fellow contributors with kindness and constructive feedback.

---

## How Can I Contribute?

### Reporting Bugs

Before creating a bug report, check the existing issues to ensure it hasn't already been reported. When filing a bug report:

1. **Use a clear title** that describes the issue concisely.
2. **Provide reproduction steps** explaining what you did, what you expected to happen, and what actually occurred.
3. **Include relevant context**, such as whether the issue happened in a Discord guild or DM, and any sanitized log output if applicable.

### Suggesting Enhancements

Feature requests and improvements are always appreciated:

1. Search open issues to verify that the idea hasn't been proposed yet.
2. Detail the motivation behind the suggestion and how it enhances the user experience.
3. For character or prompt adjustments, explain how the proposed change aligns with Columbina's canon lore and persona.

### Submitting Pull Requests

1. Fork the repository and clone your fork locally.
2. Create a dedicated branch for your changes from `main`.
3. Keep changes focused and atomic; avoid bundling unrelated changes into a single pull request.
4. Test your changes locally before submitting.
5. Open a pull request with a clear description of the problem solved and the approach taken.

---

## Local Development Setup

### Prerequisites

- [Go](https://go.dev/) (1.24 or higher recommended)
- [Git](https://git-scm.com/)
- A Discord Bot Application created via the [Discord Developer Portal](https://discord.com/developers/applications) with:
  - `Message Content Intent` enabled
  - `Server Members Intent` enabled

### Setting Up the Project

1. Clone your fork:
   ```bash
   git clone https://github.com/chromavisionmusic/columbina.git
   cd columbina
   ```

2. Download Go dependencies:
   ```bash
   go mod download
   ```

### Environment Configuration

1. Copy `.env.example` to create `.env.development`:
   ```bash
   cp .env.example .env.development
   ```

2. Populate the required environment variables:
   - `BOT_TOKEN`: Your Discord bot token.
   - `LLM_BASE_URL` & `LLM_API_KEY`: Endpoint and credentials for your LLM provider.
   - `LLM_MODEL`: Model identifier used for responses.
   - `TAVILY_API_KEY`: (Optional) API key for real-time web search capabilities.

### Running the Bot

Using `make`:

```bash
# Run in development mode
make dev

# Build the bot binary
make build

# Run tests
make test

# Run tests with race detector and coverage
make test-race

# Check and fix code formatting
make fmt

# Run static analysis and linting
make lint
```

Or using the Go toolchain directly:

```bash
# Run directly in development
go run cmd/main.go development
```

---

## Development Guidelines

### Code Style & Formatting

- Follow standard Go formatting with `gofmt -s` (`make fmt`).
- Ensure code passes `golangci-lint` (`make lint`).
- Run `go vet ./...` to check for common mistakes.
- Write readable, maintainable, and idiomatic Go code.

### Persona & Prompt Integrity

When modifying files in `prompts/` (such as `system_prompt.md`):

- Preserve Columbina's established voice: serene, polite, curious, and sincere without theatrical cruelty or caricature.
- Ensure all instructions comply with grounding, safety, and Discord formatting constraints.

### Branch Naming & Commit Conventions

Use descriptive branch names:

- `feat/feature-name` for new features
- `fix/bug-description` for bug fixes
- `docs/documentation-update` for documentation changes

We encourage clean commit messages following Conventional Commits format:

```text
feat: add slash command for user preferences
fix: resolve reply chain traversal loop guard
docs: update setup instructions in contributing guide
```

---

## Pull Request Checklist

Before opening your pull request, please ensure:

- [ ] Code compiles without errors (`go build ./...` or `make build`).
- [ ] Code is formatted with `gofmt` (`make fmt` or `gofmt -l -s .`).
- [ ] Linters pass cleanly without warnings (`make lint` or `golangci-lint run`).
- [ ] Tests pass locally (`make test` or `go test ./...`).
- [ ] No secrets, private credentials, or live API tokens are committed.
- [ ] All GitHub Actions CI checks (`.github/workflows/ci.yml`) pass.
- [ ] The PR description clearly explains the changes and references any related issue.
