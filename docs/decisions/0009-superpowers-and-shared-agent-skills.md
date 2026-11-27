# ADR-0009: Superpowers workflow and agent skills shared by every agent

## Status
Accepted

## Date
2026-11-27

## Context
Most code is written by AI agents, and contributors use different agents: Claude Code, Codex, Cursor,
Gemini CLI, Copilot and OpenCode. They should follow the same spec-driven workflow and the same domain
skills.

## Decision
- **Superpowers is the workflow:** brainstorming → spec → plan when needed → implementation → review.
  - Claude Code gets it from the plugin enabled in `.claude/settings.json`.
  - Other agents load its skills from `.agents/skills/` and start with `using-superpowers`, as AGENTS.md tells them.
- **Project skills** (TDD, Go, MCP, Postgres, Turborepo, CI hardening, ADRs, Docker Compose):
  - they live in `.agents/skills/`, the shared location other agents read, and are pinned in `skills-lock.json`;
  - `.claude/skills/` holds links to them, superpowers excluded so Claude does not load it twice.
- `AGENTS.md` is the single manual, and `CLAUDE.md` imports it.

## Alternatives considered
- **Claude-only `.claude/skills/`:** other agents would not see the skills.
- **Not vendoring skills (restore from the lockfile):** clones would not work until a restore command runs.

## Consequences
- Every agent gets the same skills on clone.
- Skill updates are reviewed like dependencies.
- Third-party skill licenses are listed in `THIRD_PARTY_NOTICES.md`.
