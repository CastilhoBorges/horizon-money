# Glossary

The product speaks Brazilian Portuguese first; the code speaks English. Use the **code** column for
identifiers, API fields, MCP tool names and docs.

## Money and movements

| pt-BR | code | Meaning |
|---|---|---|
| perfil | `user` | A person with a login |
| workspace | `workspace` | One balance in one currency |
| saldo inicial | `initial_balance` | The balance at the start of the workspace's start date |
| data inicial | `initial_balance_date` (D0) | The first day the workspace tracks |
| movimentação | `movement` | An income, expense, daily, saving or card entry |
| entrada | `income` | Money in |
| saída | `expense` | Fixed costs, bills, rent |
| diário | `daily` | Variable day-to-day spending |
| economia | `saving` | Money set aside (reserve, investment); leaves the balance |
| gasto com cartão | `card` | A card purchase or a whole card bill |
| transferência | `transfer` | Money moved between two workspaces of the same user |
| tag | `tag` | A label on movements; never changes a calculation |
| dia conferido | `checked day` | Manual "I reviewed this day" mark; display only |

## Repetition

| pt-BR | code | Meaning |
|---|---|---|
| repetição | `repetition` | none, daily, weekly, monthly or installments |
| série | `series` | The rule that produces repeated occurrences |
| ocorrência | `occurrence` | One dated instance of a series |
| parcelado | `installments` | A total split into N monthly parts |
| parcela | `part` | One installment, numbered "3/12" |
| dia âncora | `anchor day` | The day of the month a monthly series falls on; the last day in shorter months |
| escopo da edição | `scope` | `this`, `following` or `all` occurrences |
| grupo da série | `series group` | The series a `following` split produced, edited together by `all` |

## Forecast and calculations

| pt-BR | code | Meaning |
|---|---|---|
| previsão de diário | `daily forecast` | Expected daily spending applied to future days |
| divisor | `forecast_divisor` | Days (28–31) the monthly forecast is divided by |
| horizonte de saldos | `horizon` | Day-by-day projected balance over several months |
| performance | `performance` | Income minus every outflow and the remaining forecast |
| economizado | `savings rate` | Savings ÷ income |
| custo de vida | `cost of living` | Expense + daily + card + remaining forecast |
| diário médio | `average daily` | Daily spending ÷ elapsed days |
| visão consolidada | `consolidated view` | Every workspace's balance in the main currency |
| cotação | `rate` | BCB PTAX closing rate, cached per currency and date |
| — | `ledger` | A workspace's settings, movements and transfers as the domain reads them to compute views |

## Platform

| Term | Meaning |
|---|---|
| connector | Horizon Money as seen from an AI agent: the MCP server at `/mcp` |
| personal access token (PAT) | A bearer token a user creates for scripts and developer tools |
| account pages | Server-rendered sign-up, login, consent, onboarding and "connect your agent" pages |
