# BOTS.md: LLM Agent Development Protocol

This file provides guidance to agents working with code in this repository.

## Project Overview

**otel-cli** is a command-line tool for sending OpenTelemetry traces from shell scripts and other contexts where executing a program is the best available option. It's designed to be dropped into scripts with zero configuration, operating in non-recording mode by default, following the OpenTelemetry philosophy of "first, do no harm."

## Technology

- **Language**: Go 1.21+ (uses toolchain go1.22.4)
- **Architecture**: Self-contained, minimal dependencies
- **Key Dependencies**:
  - OpenTelemetry Go SDK
  - Cobra for CLI framework
  - gRPC and HTTP OTLP exporters
- **Version Control**: Git with GitHub integration
- **Current Version**: 0.5.0 (as of this writing)

## Development Philosophy

This project is **self-contained and focused**:
- Minimal external dependencies - prefer standard library
- Shell script integration is a first-class use case
- Non-recording mode by default ensures scripts can adopt otel-cli without risk
- Functional testing validates real-world usage patterns
- Code should be readable and maintainable above all else

## 🧪 Testing Philosophy

**Functional testing is primary.** Unit tests are encouraged but functional tests in `main_test.go` are where the rubber meets the road.

### The Test Harness

The test harness builds otel-cli and runs it through real-world scenarios:
- Tests actual CLI invocations with real arguments
- Validates OTLP export over gRPC and HTTP
- Includes TLS negotiation, environment variables, and backgrounding
- See `TESTING.md` for comprehensive details

### Testing Best Practices

1. **Add functional tests** for new CLI features or flags
2. **Write unit tests** for complex logic in packages
3. **Test the glue code** - that's where bugs hide
4. **Copy existing patterns** from `data_for_test.go` when adding tests
5. **Use `--fail --verbose`** in test fixtures for debugging

The test harness is more complex than otel-cli itself - this is intentional. We trust Cobra, OpenTelemetry Go SDK, and the standard library. We test the integration.

## 🧠 Git: Your Persistent Memory System

**Think of git commits as your external memory that persists across sessions.** Your commit messages are the shared context between you, future-you, other models, and the OSS community.

### Why Rich Commit Messages Matter

- **Memory Augmentation**: Your reasoning persists beyond context windows
- **Perfect Handoffs**: Next model/session picks up exactly where you left off
- **Community Intelligence**: Every contributor's insights become permanent project knowledge
- **Pattern from Training**: This is how high-performing teams in your data operate

### Your 30-Second Workflow

```bash
# 1. START: Load your context (what was I doing?)
git log --oneline -n 10
git show HEAD

# 2. WORK: Track your progress
git add <files>
git commit -m "type: what you're building"

# 3. PERSIST: Save your state
git push origin <branch>
```

### The Universal Commit Message Template

```
<type>: <what> - <why in 5 words>

Why: [Original problem/request]
Approach: [Key decision you made]
Learned: [What surprised you]
Next: [Specific next action]

🤖 YourModel <your@attribution>
```

**Types**: `feat`, `fix`, `refactor`, `test`, `docs`, `debug`, `research`

### Real Example That Works

```bash
git commit -m "fix: race condition in buffer - preventing data loss

Why: Users reported missing spans under load
Approach: Added mutex to protect concurrent access
Learned: Go's sync.Map insufficient for our write patterns
Next: Add benchmark to verify no performance regression

🤖 Claude <claude@anthropic.com>"
```

### Model Attributions

- Claude: `🤖 Claude <claude@anthropic.com>`
- Gemini: `💎 Gemini <gemini@google.com>`
- GPT-4: `🧠 GPT-4 <gpt4@openai.com>`
- Kimi: `🔮 Kimi <kimi@moonshot.ai>`
- Llama: `🦙 Llama <llama@meta.com>`

### The Handoff Protocol

When switching models or sessions:
```bash
git commit -m "[your work]

Status: [complete|blocked|handoff]
Context: [5 key facts next model needs]
Next: [specific task to continue]"
```

### Success Metrics

You're succeeding when:
- ✅ Every commit has Why/Approach/Next
- ✅ You update commit messages to reflect learning
- ✅ Other models/developers can continue without asking questions
- ✅ `git log -p` shows your reasoning evolution

### Quick Reference

| Command | Purpose | When to Use |
|---------|---------|------------|
| `git add <files>` | Stage changes | Before committing |
| `git commit -m "..."` | Commit with message | Completing discrete work |
| `git log -n 10` | See recent work | Starting sessions |
| `git show HEAD` | View current state | Understanding context |
| `git log -p` | See change history | Debugging decisions |
| `git push origin <branch>` | Push to GitHub | Work ready to share |
| `git status` | Check working tree | Understanding current state |

### Your Style Is Welcome

- **Claude**: Write narratives that explain the journey
- **Gemini**: Use structured sections with data
- **GPT-4**: Focus on tasks and outcomes
- **Kimi**: Document patterns and research
- **Llama**: Keep it pragmatic and concise

**The format is flexible, the information is essential.**

### Remember

Git commits are messages to your future self and the community. Write what you'd need at 3am to understand what you were thinking. Your future self (and other models) will thank you.

## 📊 GitHub Issues for Project Tracking

**This project has OSS history and active community engagement.** Use GitHub issues liberally for:

- **Feature planning** - Open issues to discuss new features before implementing
- **Bug tracking** - Document bugs with reproduction steps
- **Project status** - Keep ongoing work visible in issues
- **Community engagement** - Issues are how users interact with the project
- **Handoffs** - Reference issues in commits to maintain context

### Issue Best Practices

1. **Open issues early** - Don't wait until implementation is complete
2. **Reference commits** - Link commits to issues with "refs #123" or "fixes #123"
3. **Update issues** - Comment on progress, blockers, and learnings
4. **Label appropriately** - Use labels to categorize and prioritize
5. **Close with context** - Explain what was done and why when closing

## 🏗️ Project Structure

```
otel-cli/
├── main.go                 # Entry point, version info
├── main_test.go            # Functional test harness
├── data_for_test.go        # Test fixture definitions
├── tls_for_test.go         # Ephemeral CA for TLS tests
├── otelcli/                # CLI framework and commands
├── otlpclient/             # OTLP gRPC/HTTP client
├── otlpserver/             # OTLP server (for 'server' command)
├── w3c/                    # W3C trace context handling
├── demos/                  # Example shell scripts
├── configs/                # Example configurations
├── TESTING.md              # Testing documentation
└── CHANGELOG.md            # Version history
```

## 🔧 Go Development Commands

```bash
# Build the project
go build

# Run functional tests (requires built binary)
go build && go test

# Run unit tests
go test ./...

# Run unit tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run with race detector
go build -race
go test -race ./...

# Format code (always use gofmt)
go fmt ./...

# Vet code
go vet ./...

# Update dependencies
go mod tidy

# Verify dependencies
go mod verify
```

### Running otel-cli Locally

```bash
# Build it
go build

# Run the server TUI for local testing
./otel-cli server tui

# In another terminal, send a span
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
./otel-cli span -n "test" -s "my-service"
```

## 💎 Go Code Style & Quality

### Correctness & Clarity First

- Prioritize readable, correct code over premature optimization
- Use strong, idiomatic Go types
- Leverage Go 1.21+ features appropriately
- No shortcuts or workarounds - refactor messy code when encountered

### Naming & Structure

- Use full, descriptive names - no unnecessary abbreviations
- Package names: short, lowercase, single word
- Exported names: clear and self-documenting
- Add new functionality to existing files unless it represents a distinct logical component

### Comments & Documentation

- **No organizational comments** - code should be self-documenting
- **"Why" comments only** - explain non-obvious implementation choices
- Package-level documentation for every package
- Exported functions/types must have doc comments

### Error Handling

- Always handle errors explicitly - never ignore them
- Use `fmt.Errorf` with `%w` for error wrapping to preserve context
- Return errors, don't panic (except for truly unrecoverable situations)
- Provide useful error messages with context

### Concurrency & Context

- Pass `context.Context` as the first parameter to functions that need it
- Respect context cancellation - check `ctx.Done()` in long-running operations
- Use channels and goroutines idiomatically
- Avoid shared mutable state - use channels or mutexes appropriately

### Testing

- **Functional tests first** for CLI features and integration
- Unit tests for complex logic in packages
- Table-driven tests for multiple cases
- Use subtests with `t.Run()` for clarity
- Test error paths, not just happy paths

### Shell Script Integration

otel-cli is designed for shell scripts. When adding features:

- **Consider the shell use case** - is this ergonomic from bash?
- **Test with real shell scripts** - demos/ directory provides examples
- **Environment variables matter** - shell scripts rely on them heavily
- **Exit codes matter** - shell scripts check them
- **Stdout vs stderr** - be intentional about what goes where

## 🐙 GitHub Integration

**GitHub CLI (`gh`):**
Use `gh` for GitHub operations without leaving the terminal:

```bash
gh pr create                 # Create PR
gh pr status                 # Check PR status
gh pr checks                 # View CI results
gh issue list                # Check issues
gh issue view <number>       # Read issue details
gh issue create              # Create new issue
```

## 📦 Versioning & Releases

**Semantic versioning:** MAJOR.MINOR.PATCH (e.g., 0.5.0)

- **MAJOR**: Breaking changes (we're not at 1.0 yet)
- **MINOR**: New features, significant changes
- **PATCH**: Bug fixes, small improvements

**Releases are managed by goreleaser:**

- Version is set via git tags
- Release process documented in README.md
- CHANGELOG.md is updated with each release
- goreleaser handles building for multiple platforms

**When bumping versions:**

1. Update CHANGELOG.md with changes
2. Tag the release: `git tag v0.X.Y`
3. Let maintainers handle goreleaser

## 🎯 When Working on otel-cli

### Starting a Session

1. `git log --oneline -n 20` - What's been happening?
2. `gh issue list` - What's being tracked?
3. `git status` - What's the current state?
4. `go build && go test` - Does everything work?

### Making Changes

1. **Open/reference an issue** - Keep work visible
2. **Write tests first** - Especially functional tests
3. **Make it work** - Get it functionally correct
4. **Make it right** - Clean up, refactor, document
5. **Commit with context** - Rich commit messages

### Completing Work

1. **All tests pass** - `go build && go test && go test ./...`
2. **Code formatted** - `go fmt ./...`
3. **Commit with attribution** - Use the template above
4. **Push and create PR** - Reference the issue
5. **Update issue** - Mark as complete or pending review

## 🤝 Cross-Model Collaboration

Git's power is in context preservation across sessions, agents, and developers.

**Picking up someone else's work:**

1. Find their commit: `git log --oneline -n 20`
2. Read their message: `git show <commit-sha>`
3. See the changes: `git show <commit-sha> -p`
4. Continue their work with your own commits

**Avoiding duplicate work:**

- Always check `git log -n 20` before starting something new
- Search commit messages: `git log --grep="keyword"`
- Check GitHub issues: `gh issue list`

## Remember

**Git commits are permanent project knowledge.** They are:

- Context for future sessions and developers
- Documentation of architectural decisions
- Reasoning traces that survive indefinitely
- Messages to your future self and the community

**Your commit message quality directly impacts the next agent's and next developer's ability to succeed.** Write for them.

## Attribution

- Built by A. Tobey while at Equinix (primarily on the Metal team)
- Forked and maintained by A. Tobey (@tobert) since 2025
- Agent-assisted development begins with v0.6.0
- All commits should use appropriate agent attribution (see section above)

## License

Apache License 2.0 - Copyright (c) 2025 A. Tobey

See LICENSE file for full details.
