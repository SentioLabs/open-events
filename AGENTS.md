# Agent Instructions

This repository is **OpenEvents**, a spec-first event taxonomy compiler written in Go. The MVP validates Git-backed YAML event registries and produces a deterministic normalized model.

These instructions apply to the **compiler** codebase (`cmd/`, `internal/`). The standalone exemplar application under `examples/demo/` has its own agent instructions at `examples/demo/CLAUDE.md`; agents working only in `internal/` should not need to know the demo exists.

## Work Tracking

This project uses **arc** for issue tracking. Run `arc onboard` if arc is not initialized for your session.

Quick commands:

```bash
arc ready                         # Find available work
arc show <id>                     # View issue details
arc update <id> --take             # Claim work for this session
arc close <id>                    # Complete work
arc create "title" --type=task    # File follow-up work
```

Use arc for multi-session work, dependencies, bugs, discovered follow-ups, and anything that should survive the current session. Use the in-session todo list only for short-lived execution steps.

## Project Shape

- CLI entrypoint: `cmd/openevents/main.go`
- CLI implementation: `internal/cli/`
- Registry loading, validation, diagnostics, and model types: `internal/registry/`
- Schemair (lock model, lock update/check, registry → IR lowering): `internal/schemair/`
- Per-language emitters: `internal/codegen/golang/`, `internal/codegen/python/`
- Protobuf emitter: `internal/protogen/`
- Integration tests: `internal/integration/`

## Development Commands

Run commands from the repository root.

```bash
go test ./...
go run ./cmd/openevents validate ./examples/demo/registry
```

Expected successful validation output is similar to:

```text
ok: registry valid (12 events across 2 domains)
```

Before committing Go changes, run `gofmt` on modified Go files and then `go test ./...`.

## Implementation Guidelines

- Keep the compiler deterministic: stable ordering, predictable diagnostics, and no hidden filesystem-order dependencies.
- Prefer small, focused packages and explicit data structures over broad abstractions.
- Preserve the MVP scope in `README.md`: validation first, later milestones for snapshots, diffs, codegen, schemas, Snowflake, and docs.
- Add or update tests with behavior changes. Favor table-driven tests for validation rules.
- When changing registry semantics, update examples and documentation if user-visible behavior changes.
- Keep error messages actionable and tied to registry paths, event names, or field names when possible.

## YAML Registry Expectations

- Registry examples live under `examples/` and `internal/registry/testdata/`.
- Test data should be minimal and named for the behavior under test.
- Avoid introducing generated artifacts into the repository unless a task explicitly requires them.

## Landing the Plane (Session Completion)

When ending a work session, complete all applicable steps below.

1. **File issues for remaining work** - Create arc issues for follow-ups, bugs, or deferred cleanup.
2. **Run quality gates** - If code changed, run tests and relevant validation commands.
3. **Update issue status** - Close finished arc issues and update in-progress items.
4. **Commit and push**:
   ```bash
   git status
   git add <files>
   git commit -m "description of changes"
   git push
   git status
   ```
5. **Verify** - The final `git status` must show the branch is clean and up to date with origin.
6. **Hand off** - Summarize what changed, what was verified, and any remaining work.

Critical rules:

- Work is not complete until `git push` succeeds.
- Do not leave completed work only in the local worktree.
- Do not say "ready to push when you are"; push the completed work.
- If push fails, resolve the issue and retry until it succeeds or clearly report the blocker.
