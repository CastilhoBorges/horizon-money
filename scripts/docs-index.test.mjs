import assert from "node:assert/strict";
import { test } from "node:test";

import {
  findOversizeDocs,
  parseAdr,
  renderNotices,
  replaceIndex,
} from "./docs-index.mjs";

test("parseAdr reads the number, title and status", () => {
  const adr = parseAdr(
    "0007-turborepo.md",
    "# ADR-0007: Turborepo and pnpm monorepo\n\n## Status\nAccepted\n\n## Date\n2026-11-27\n",
  );
  assert.deepEqual(adr, {
    file: "0007-turborepo.md",
    number: "0007",
    title: "Turborepo and pnpm monorepo",
    status: "Accepted",
  });
});

test("parseAdr keeps a superseded status line as written", () => {
  const adr = parseAdr("0002-x.md", "# ADR-0002: X\n\n## Status\nSuperseded by ADR-0009\n");
  assert.equal(adr.status, "Superseded by ADR-0009");
});

test("replaceIndex swaps only the content between the markers", () => {
  const before = "# Title\n\nIntro.\n\n<!-- index:start -->\nold\n<!-- index:end -->\n\nFooter.\n";
  const after = replaceIndex(before, "new line");
  assert.equal(
    after,
    "# Title\n\nIntro.\n\n<!-- index:start -->\nnew line\n<!-- index:end -->\n\nFooter.\n",
  );
});

test("replaceIndex appends markers when a README has none", () => {
  assert.equal(
    replaceIndex("# Title\n", "x"),
    "# Title\n\n<!-- index:start -->\nx\n<!-- index:end -->\n",
  );
});

test("renderNotices lists each source once with its license and commit", () => {
  const lock = {
    skills: {
      tdd: { source: "mattpocock/skills", commit: "abc1234def" },
      "golang-lint": { source: "samber/cc-skills-golang", commit: "fff0000aaa" },
      "golang-naming": { source: "samber/cc-skills-golang", commit: "fff0000aaa" },
    },
  };
  const licenses = { "mattpocock/skills": "MIT", "samber/cc-skills-golang": "MIT" };
  const text = renderNotices(lock, licenses);
  assert.match(text, /\| \[mattpocock\/skills\]\(https:\/\/github\.com\/mattpocock\/skills\) \| MIT \| tdd \| `abc1234` \|/);
  assert.match(text, /\| \[samber\/cc-skills-golang\][^\n]*\| MIT \| golang-lint, golang-naming \| `fff0000` \|/);
});

test("renderNotices refuses a source with no known license", () => {
  const lock = { skills: { x: { source: "someone/unknown", commit: "1234567" } } };
  assert.throws(() => renderNotices(lock, {}), /someone\/unknown/);
});

test("findOversizeDocs reports files over the line limit", () => {
  const files = { "docs/PROJECT.md": "a\n".repeat(10), "docs/GLOSSARY.md": "a\n".repeat(301) };
  assert.deepEqual(findOversizeDocs(files, 300), [{ path: "docs/GLOSSARY.md", lines: 301 }]);
});
