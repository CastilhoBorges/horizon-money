# Community setup

**Date:** 2026-11-27

## What happened

- **Community files:** CONTRIBUTING (including the AI-assisted contributions policy), the Contributor
  Covenant code of conduct, a SECURITY policy that uses GitHub private vulnerability reporting, issue
  forms, a PR template that mirrors the Definition of Done, and CODEOWNERS.
- **Dependabot** covers Go modules, npm, GitHub Actions, the Dockerfile and `compose.yaml`. Commit
  prefixes (`build`, `ci`) keep its PR titles valid Conventional Commits.
- **Code analysis:** OpenSSF Scorecard runs on `main`, weekly and on branch-protection changes, and
  uploads SARIF to code scanning. CodeQL default setup is on.
- **Repo files:** `.editorconfig` and `.gitattributes`. Vendored skills are marked `linguist-vendored`
  so they do not skew the language statistics.

## Learned

- release-please moved to the release-pipeline issue. PRs opened with the default `GITHUB_TOKEN` do not
  trigger CI, so release PRs would never satisfy the required checks. That needs a GitHub App or
  fine-grained token, which belongs with the release work.
