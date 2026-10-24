# otel-cli agent guide

otel-cli sends OpenTelemetry spans from shell scripts and other places where a program
call is the best option. It is one Go binary that needs no configuration to run.

Scripts depend on this behavior. Do not change it without a maintainer decision:

- With no endpoint, otel-cli is in non-recording mode. It exports nothing and exits 0.
- An export error does not change the exit code. otel-cli exits 0 and prints nothing.
- `--fail` (`OTEL_CLI_FAIL`) makes an export error exit 1.
- `--verbose` (`OTEL_CLI_VERBOSE`) prints errors to stderr.
- `otel-cli exec` exits with the exit code of the child, also when the export fails.

## Writing style

This style applies to documentation, `--help` text, error messages, code comments,
commit messages, PR descriptions and CHANGELOG.md.

The style is based on ASD-STE100 Simplified Technical English (STE). We use these
rules, not the full STE dictionary:

- Write one topic in each sentence. Keep an instruction to 20 words or fewer and a
  description to 25 words or fewer.
- Use the imperative for instructions: "Run `go build` before `go test`."
- Use the active voice and name the actor: "otel-cli exits 1", not "1 is returned".
- Use common words. Do not use idioms, figures of speech or jokes. Some readers have
  English as a second language, and some readers have only part of the context.
- Use one term for one concept. Do not change words for variety. The Terms table is
  the source.
- Give specific values: the flag, the environment variable, the default, the exit code.
- Put the most important information first. The reader can lose the end of the text.
- Show the correct example before the explanation. If you must show an incorrect
  example, put the correct one first and mark the error.
- Use American spelling.
- Do not add decoration: no emoji headings, no marketing words.

> Before: The span might not get sent if something goes wrong with the connection.
>
> After: If the export fails, otel-cli sends nothing and exits 0. Add `--fail` to exit 1.

### Each kind of text

**Documentation and help text.** This is README.md, TESTING.md, `demos/`, and the Cobra
`Short`, `Long` and flag descriptions. Users and agents read help text. Apply all of
the rules.

**Error messages.** Give the value, the rule and the fix. Do not show internal names
that the user cannot act on. Wrap Go errors with `fmt.Errorf("context: %w", err)`.

**Code comments.** Write why, not what. Use one line when one line is enough. Give a
reason one time in each file, not on each repeated step. Put the narrative in the
commit message.

**Commit messages.** Use the template in "Version control". Write 1 to 3 sentences in
each field. Quote the user when their request is the reason for the change. Give the
evidence: commands, versions, CI run IDs.

**PR descriptions.** The PR description becomes the squash commit message. Write it as
a commit message. Start with what changed and why. List the verification commands. Do
not repeat content in different sections.

**CHANGELOG.md.** Write one line for each bullet. Name the change, not the
investigation.

### Terms

Terms from OpenTelemetry and W3C Trace Context have their specification meaning. This
table has only the terms that otel-cli adds. Add a term when two meanings collide in
real text, not before.

| Term | Meaning |
|---|---|
| non-recording mode | No endpoint is set. otel-cli runs normally but exports nothing. |
| soft fail | An error path that exits 0, or 1 with `--fail`. In code: `Config.SoftFail`. |
| fixture | One test case in `data_for_test.go`. |
| functional test | A test that runs the built `./otel-cli` against a test OTLP server. |

## Project structure

```
otel-cli/
├── main.go             # Entry point and version information
├── main_test.go        # Functional test harness
├── data_for_test.go    # Functional test fixtures
├── tls_for_test.go     # Temporary CA for the TLS tests
├── otelcli/            # Cobra commands and configuration
├── otlpclient/         # OTLP gRPC and HTTP client
├── otlpserver/         # OTLP server for `otel-cli server` and the tests
├── w3c/                # W3C trace context
├── demos/              # Example shell scripts
├── configs/            # Example configurations
├── release/            # Release image Dockerfile
├── .github/workflows/  # CI and release workflows
├── TESTING.md          # Test harness documentation
└── CHANGELOG.md        # Version history
```

## Gates

CI runs these on every PR. Run them before you push.

```bash
go build && go test ./...
go vet ./...
gofmt -l .                                  # prints nothing when clean
go mod tidy && git diff --exit-code go.mod go.sum
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
```

CI also builds a GoReleaser snapshot.

### Hazards

- The functional tests run `./otel-cli`. Run `go build` first, or the tests use an old
  binary.
- The harness does not compare the `ExitCode` of a fixture. Check exit codes in a
  `CheckFuncs` entry.
- `Config.SoftFail` calls `os.Exit`. Code after a `SoftFail` call does not run.
- Parse integer values from users in base 10: `strconv.ParseInt(v, 10, 64)`. Base 0
  reads `0x10` as 16 and `010` as 8.
- CI and releases build with the latest stable Go. The `go` line in go.mod is only the
  minimum version.
- The `build` job in `ci.yml` is a required status check. If you rename it, every PR
  is blocked.

## Testing

Functional tests are the primary tests. TESTING.md documents the harness.

- Add a functional test for each new CLI feature or flag. Copy a fixture from
  `data_for_test.go`.
- Add unit tests for complex logic in packages. Use table-driven tests and `t.Run`.
- Test the error paths, not only the success paths.
- Add `--fail --verbose` to a fixture to see its errors.

We trust Cobra, the OpenTelemetry Go SDK and the standard library. We test the
integration between them, because most bugs are there.

## Go code

- Write correct, readable code first. Optimize only after a measurement.
- Prefer the standard library. Add a dependency only when it removes real work.
- Use full, descriptive names. Do not abbreviate.
- Give each package a package doc comment. Give each exported name a doc comment.
- Handle every error. Return errors. Panic only when the program cannot continue.
- Pass `context.Context` as the first parameter. Stop work when the context is done.
- Run `gofmt`.

### Shell scripts are the main users

- Make each feature easy to use from bash.
- Give each new configuration flag an environment variable, as `otelcli/config.go` does.
- Exit codes are part of the interface. Scripts check them.
- Write data to stdout and diagnostics to stderr.

## Version control

- `main` is protected. Each change lands through a PR, and PRs are squash-merged.
- Agents can open PRs. A human decides to merge.
- Ask a maintainer before you open an issue. Refer to issues with `refs #N` or
  `fixes #N`.
- Commit often in your branch. The commit history is the memory for the next session,
  agent or developer.

Start a session with `git log --oneline -n 20`, `gh issue list`, `git status` and
`go build && go test ./...`.

### Commit message template

```
<type>: <what> - <why, in a few words>

Why: <the problem or the request>
Approach: <the decision and the reason for it>
Learned: <what was unexpected>
Next: <the next specific action>

🤖 <Model> <attribution address>
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`, `build`, `research`.

```
fix(attrs): parse integer attributes in base 10 - stop octal coercion

Why: `--attrs n=010` exported 8, not 10 (#373).
Approach: strconv.ParseInt(v, 10, 64) instead of base 0.
Learned: base 0 also accepts the 0x, 0o and 0b prefixes.
Next: none.
```

When you stop before the work is complete, add these fields:

```
Status: complete | blocked | handoff
Context: <the facts that the next agent needs>
Next: <the task to continue>
```

### Model attribution

Credit each model that wrote or reviewed the change, with one line for each model.

## Releases

otel-cli uses semantic versioning. Before 1.0, a minor release can break compatibility.

A `v*` tag push runs `.github/workflows/release.yml`. GoReleaser (`.goreleaser.yml`)
makes the binaries, packages, SBOM, signatures, GitHub release and Homebrew formula.
The workflow then builds, signs, attests and tags the ghcr.io image.

- A `-rc.N` tag makes a GitHub prerelease. It does not update the Homebrew tap or
  `latest`.
- A `workflow_dispatch` run is a snapshot. It publishes nothing.

To release:

1. Update CHANGELOG.md and merge it to main.
2. Tag main: `git tag v0.X.Y && git push origin v0.X.Y`.
3. Watch the `release` workflow.

README.md, "Releases" has the full procedure and the required secret.

## Attribution

- A. Tobey wrote otel-cli at Equinix, mostly on the Metal team.
- A. Tobey (@tobert) forked it and maintains it since 2025.
- Agent-assisted development starts with v0.6.0.

## License

Apache License 2.0. Copyright (c) 2025 A. Tobey. See LICENSE.
