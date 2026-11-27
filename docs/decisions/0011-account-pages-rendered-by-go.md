# ADR-0011: Account pages rendered by the Go server

## Status
Accepted

## Date
2026-11-27

## Context
OAuth needs pages: sign-up, login, Google sign-in, consent and onboarding, ending with "connect your
agent". The full web app comes after v0.1, but the connector must be usable from v0.1.

## Decision
The Go server renders these pages with `html/template`:
- no frontend framework;
- pt-BR and en;
- a strict Content-Security-Policy and CSRF-protected forms.

When the Next.js app arrives, it signs users in through these pages, the way an app uses "sign in with …".

## Alternatives considered
- **Build the pages in Next.js early:** nicer pages sooner, but v0.1 would depend on two apps, and the
  OAuth flow would be split across them.

## Consequences
- Sign-in and consent live in one place, next to the authorization server.
- v0.1 is complete without the web app: sign up, connect an agent and use everything through it.
- The pages stay minimal until the web app restyles the experience.
