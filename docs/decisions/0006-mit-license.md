# ADR-0006: MIT license

## Status
Accepted

## Date
2026-11-27

## Context
The project is meant to be used, self-hosted and improved by as many people as possible.

## Decision
License the code under MIT. Third-party material bundled in the repository, such as the agent skills,
keeps its own license, and those licenses are listed in `THIRD_PARTY_NOTICES.md`.

## Alternatives considered
- **AGPL-3.0:** forces anyone offering a modified version as a service to publish it, but it puts off
  some contributors and users.
- **Apache-2.0:** permissive and includes a patent grant, but it is more text for the same practical outcome here.

## Consequences
- Anyone may use, modify and redistribute the code, including in closed products.
- Bundled Apache-2.0 skills keep their LICENSE files and are listed in the notices.
