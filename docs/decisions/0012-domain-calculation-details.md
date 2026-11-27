# ADR-0012: Calculation details the domain spec leaves open

## Status
Accepted

## Date
2026-11-27

## Context
The product spec (§4) defines the domain rules, but a few edges are not exact enough to test. These are
the status boundary, how rates round, what happens around D0, and how series edits keep their shape.
`internal/domain` needs one answer for each before HTTP and MCP expose them.

## Decision
1. **Low status.** A day is `low` when `0 ≤ end < low_balance_threshold`. With the default threshold of 0,
   a day is never `low`.
2. **Savings rate.** It is stored in basis points (27.14% = `2714`), rounded half-up. The target band is
   compared on that rounded value, so the label always matches the number shown.
3. **Average daily.** It is rounded half-up to the minor unit.
4. **D0.** The balance table never returns days before D0. Any balance before D0 is the initial balance.
5. **Occurrence overrides.** An override is a full snapshot (date and amount) of one occurrence, keyed by
   the date the series produced. A later `all` edit never touches it.
6. **Anchor day.** Monthly series and installments store the day of the month they fall on. A `following`
   split keeps it, so a series on the 31st stays on the 31st after February.
7. **Splitting at the first occurrence** is refused (`series.split_at_first_occurrence`). The use case
   edits the series instead.
8. **Horizon.** It spans 1 to 12 months, 3 by default.
9. **Installment parts.** A total below one minor unit per part is refused
   (`series.installment_part_too_small`), so no part is ever zero.

## Alternatives considered
- **`low` up to and including the threshold:** with the default of 0, every zero balance would be `low`,
  which leaves no way to turn the warning off.
- **Comparing the savings band on the exact ratio:** a rate shown as 30.00% could then be labelled
  "above" a 30% bound.
- **Field-level overrides:** they leave it unclear which fields an `all` edit may still change on an
  edited occurrence.

## Consequences
- Each decision has a test in `internal/domain`.
- HTTP and MCP return these values as they are, so both doors agree.
- Changing one of them needs a new ADR that supersedes this one.
