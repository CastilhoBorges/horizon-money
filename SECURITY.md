# Security policy

Horizon Money stores personal financial data and issues OAuth tokens to AI agents, so security reports
are taken seriously.

## Reporting a vulnerability

**Do not open a public issue.** Report privately through GitHub:

1. Go to [Security → Report a vulnerability](https://github.com/CastilhoBorges/horizon-money/security/advisories/new).
2. Describe the issue, the affected version or commit, and the steps to reproduce it.

You should get a first answer within 7 days. We will agree a disclosure date with you, credit you in the
advisory unless you prefer otherwise, and publish a fix before details become public.

## Supported versions

The project is pre-1.0. Only the latest release and `main` receive security fixes.

## Scope

In scope:
- **Authentication and authorization:** sessions, personal access tokens, the OAuth 2.1 authorization
  server, Google sign-in, and password reset.
- **Data isolation:** one user reaching another user's data.
- **The MCP server and HTTP API:** injection, SSRF, and anything that leaks data in responses or logs.
- **The container image and the default `compose.yaml`.**

Out of scope:
- Vulnerabilities in a self-hoster's own reverse proxy, tunnel or host configuration.
- Findings that need a compromised server or a malicious administrator.

## Hardening already in place

- CI actions are pinned by commit SHA and run with least-privilege permissions.
- Dependabot keeps dependencies, actions and images updated.
- govulncheck runs on every PR.
- Secret scanning with push protection is enabled.
- OpenSSF Scorecard runs on the default branch.
