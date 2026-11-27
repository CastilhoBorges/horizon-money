# Contributing to Horizon Money

Thanks for helping. This project is built largely with AI coding agents, so the rules are written down
and checked by machines. Read [`AGENTS.md`](AGENTS.md) first. It is the manual for people and agents alike.

## Ways to help

- **Report a bug or suggest a feature** with the issue forms.
- **Ask a question or share an idea** in [Discussions](https://github.com/CastilhoBorges/horizon-money/discussions).
- **Pick an issue** labelled `good first issue` or `help wanted`, and comment that you are taking it.
- **Report a vulnerability privately**. See [`SECURITY.md`](SECURITY.md). Never open a public issue for one.

## Set up

You need Go (the version in `apps/api/go.mod`), Node 24 or newer, pnpm and Docker.

```bash
git clone https://github.com/CastilhoBorges/horizon-money.git
cd horizon-money
pnpm install          # tooling + git hooks (lefthook)
cp .env.example .env
pnpm db:up            # Postgres + Mailpit
pnpm test && pnpm lint
pnpm dev
```

## How work flows

1. **Start from an issue.** For anything larger than a small fix, agree the approach in the issue first.
   New behaviour gets a short design in `docs/specs/`.
2. **Use a worktree and a branch:** `git worktree add -b feat/<slug> .claude/worktrees/feat-<slug> main`.
3. **Write the test first** and watch it fail, then make it pass. Never edit a test just to make it pass.
4. **Keep doors thin.** HTTP handlers and MCP tools call one use case. Business rules live in `internal/domain` and `internal/app`.
5. **Commit with Conventional Commits** (`feat(api): add tags`). The commit-msg hook checks them.
6. **Open a PR** whose title is a Conventional Commit, fill in the template, and link the issue (`Closes #N`).
   PRs are squash-merged once CI is green.

The full checklist is the Definition of Done in [`AGENTS.md`](AGENTS.md).

## Using an AI coding agent

You are welcome to use one. The repository is set up for it:
- **Shared skills** live in `.agents/skills/`. Codex, Cursor, Gemini CLI, Copilot, OpenCode and others read them there.
- **Claude Code** reads them through `.claude/skills/`, and offers the superpowers plugin enabled in `.claude/settings.json`.
- **Other agents** should load the `using-superpowers` skill before starting. It explains the spec-driven workflow the project follows.
- **Windows without symlinks:** if `.claude/skills/` arrives as plain text files, run
  `npx skills experimental_install -a claude-code universal`.

**AI-assisted contributions policy:**
- Say in the PR that you used an agent, and which one.
- The person who opens the PR must understand every line and answer review questions about it. "The AI wrote it" is not an answer.
- Don't open PRs you have not run locally (`pnpm test`, `pnpm lint`).
- Never paste secrets, private data or third-party code you have no right to share into a prompt or a PR.

## Docs

Docs grow sideways (see ADR-0008):
- one new file per decision in `docs/decisions/`;
- one new file per devlog entry in `docs/devlog/`;
- living docs stay under about 300 lines.

Run `pnpm docs:index` after adding a doc, and CI checks the result.

## License

By contributing, you agree that your contributions are licensed under the [MIT License](LICENSE).
