# ADR-0010: Built-in OAuth 2.1 authorization server for MCP

## Status
Accepted

## Date
2026-11-27

## Context
The connector must work in cloud agents such as the Claude apps, which require OAuth for remote MCP
servers. The current MCP authorization spec prefers Client ID Metadata Documents (CIMD), and widely used
clients still register through Dynamic Client Registration (DCR). Self-hosters should not have to run a
separate identity provider.

## Decision
Build the authorization server into the binary:
- protected-resource (RFC 9728) and authorization-server (RFC 8414) metadata;
- client registration through **both** CIMD and DCR;
- authorization code with PKCE `S256` required;
- short-lived access tokens, and refresh tokens that rotate on use and revoke their family on reuse;
- audience-bound tokens and exact redirect-URI matching;
- consent per user and client, revocable from the profile.

Personal access tokens remain for developer tools and scripts.

## Alternatives considered
- **Tokens only:** works in developer tools, but the connector would not appear in the Claude apps.
- **An external identity provider (Keycloak, Ory):** robust, but a second service for every self-hoster.

## Consequences
- This is the most security-sensitive code in the project. It gets end-to-end flow tests and a test
  against a real client before release.
- The install needs a public HTTPS URL (`APP_BASE_URL`) for cloud agents.
