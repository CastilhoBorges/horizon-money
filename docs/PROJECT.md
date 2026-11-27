# Horizon Money — project

## Vision

Know in advance how much money you will have, without spreadsheets, and ask your AI agent about it
in plain language.

Horizon Money is an open-source, self-hosted personal-finance forecasting app. You log what comes in and
what goes out, and it projects your balance day by day for the months ahead. Its main face is an
**MCP connector**: you plug it into the agent you already use (Claude, ChatGPT, Cursor and others) and
say "I spent 30 on lunch", "how much will I have on the 15th?" or "close my month".

Horizon Money is not an agent and runs no AI model. It is what lets **your** agent understand **your** money.

## Who it is for

- People who track their money by hand and want a forecast, not just a history.
- People who keep money in more than one currency.
- People who want to own their data and run the app on their own server.

## Scope of v0.1

- **Domain:**
  - five movement types (income, expense, daily, saving, card), with recurrence and installments;
  - the daily forecast and the day-by-day balance with a multi-month horizon;
  - monthly and yearly calculations (performance, savings rate, cost of living, average daily);
  - tags and the "checked day" mark.
- **Several currencies:** workspaces with their own currency and balance, manual transfers, and a
  consolidated view at the BCB PTAX rate.
- **Connectors:**
  - an MCP server with tools and prompts;
  - OAuth 2.1, so it works as a connector in the Claude apps, plus personal access tokens;
  - an HTTP API for the future web app.
- **Accounts:** email + password or Google, server-rendered account pages, and optional SMTP for password resets.
- **Self-hosting:** `docker compose up`.

## Out of scope for now

- Bank connectors and imports.
- Exchange providers.
- Credit-card registries.
- Holidays.
- Email verification.
- A built-in AI model.
- The full web app, which comes after v0.1.

The complete design is in [`specs/`](specs/).
