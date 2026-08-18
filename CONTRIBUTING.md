# Contributing

## GitHub workflow

This repository uses a simple and consistent GitHub process for development tracking and project continuity:

- `main` is the current working branch for the active development state.
- `feature/*` is used for new work and larger improvements.
- `hotfix/*` is used for urgent fixes.
- `release/*` is used only when a stable deployment snapshot is being prepared.

Pull requests should stay focused, be reviewable, and include a clear summary and validation notes.

## Issue workflow

- Use a bug report for defects or regressions.
- Use a feature request for improvements or new capabilities.
- Add the appropriate label before or during triage.
- Include reproduction steps, expected behavior, and environment details.

## Pull request workflow

- Keep the PR narrow and directly related to a single problem.
- Add a summary, testing steps, and checklist items.
- Link the related issue when relevant.
- Request review before merging.

## Commit conventions

Use Conventional Commits:

- `feat: ...` for new features
- `fix: ...` for bug fixes
- `docs: ...` for documentation updates
- `security: ...` for security-related changes
- `refactor: ...` for code cleanup without behavior change
- `chore: ...` for maintenance and tooling
- `test: ...` for tests
- `ci: ...` for CI pipeline updates
- `release: ...` for release metadata or version bumps

Examples:

- `feat: add UPS monitoring endpoint`
- `fix: correct JWT refresh token expiry`
- `docs: add GitHub issue templates`
- `security: harden backup retention config`

Commit messages should be imperative, short, and specific.

## Versioning and beta snapshots

This project uses a lightweight versioning approach for development continuity rather than a public release pipeline:

- `main` tracks the current development state.
- Internal beta snapshots may use SemVer-style tags such as `v2.2.0-beta.1`.
- These tags are used as development checkpoints, not as a public release announcement.
- A stable public release is only created when the project is genuinely ready for broader use.

This keeps the project traceable without pretending to be in a formal public release lifecycle too early.

## Label system

Repository labels are codified in `.github/labels.yml` and synced through GitHub Actions.

Primary categories:

- `type:*` for bug, feature, security, docs, refactor, chore
- `priority:*` for low, medium, high, critical
- `status:*` for triage, in-progress, blocked, ready
- `release:*` for internal beta/stable tracking only

This keeps issues, pull requests, and project status consistent across the project without creating unnecessary release overhead.
