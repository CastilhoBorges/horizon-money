# Documentation map

Start here, then open only the file you need. Every folder that grows has a generated `README.md` index.

| Folder / file | What it holds | How it grows |
|---|---|---|
| [`PROJECT.md`](PROJECT.md) | Vision, audience and scope | Rarely changes |
| [`GLOSSARY.md`](GLOSSARY.md) | Domain terms and their names in code | Edited in place; split by domain past ~300 lines |
| [`ROADMAP.md`](ROADMAP.md) | Phases only; the detailed backlog is GitHub issues | Edited when a phase starts or ends |
| [`architecture/`](architecture/) | How the code is organised, one file per area | Edited in place; split by area |
| [`decisions/`](decisions/) | Architecture Decision Records (ADRs) | One new file per decision; never rewritten |
| [`devlog/`](devlog/) | What was done and learned, by date | One new file per entry |
| [`specs/`](specs/) | Approved designs, one per feature | One new file per design |
| [`plans/`](plans/) | Implementation plans, when a feature needs one | One new file per plan |

## Rules

- **History is append-only.** A decision that changes gets a new ADR that supersedes the old one. Only
  the old one's status line is updated.
- **Living docs stay small.** `PROJECT`, `GLOSSARY`, `ROADMAP`, `architecture/*` and every README stay
  under about 300 lines. CI fails when one grows past that.
- **Indexes are generated.** Run `pnpm docs:index` after adding a file. CI fails on a stale index.
- **The repository is public.** No personal data and no private context, ever.
