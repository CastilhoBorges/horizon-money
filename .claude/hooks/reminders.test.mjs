import assert from "node:assert/strict";
import { test } from "node:test";

import { newFileReminder, sessionContext, specSkillReminder } from "./reminders.mjs";

test("sessionContext points at the manual and the reuse rule", () => {
  const text = sessionContext();
  assert.match(text, /AGENTS\.md/);
  assert.match(text, /docs\/README\.md/);
  assert.match(text, /third copy/i);
});

test("newFileReminder fires for a new Go or TypeScript source file", () => {
  const exists = () => false;
  for (const path of ["/repo/apps/api/internal/domain/money.go", "/repo/apps/web/app/page.tsx", "/repo/x.ts"]) {
    assert.match(newFileReminder({ tool_input: { file_path: path } }, exists), /reuse/i, path);
  }
});

test("newFileReminder stays quiet for existing files and non-source files", () => {
  assert.equal(newFileReminder({ tool_input: { file_path: "/repo/a.go" } }, () => true), null);
  assert.equal(newFileReminder({ tool_input: { file_path: "/repo/README.md" } }, () => false), null);
  assert.equal(newFileReminder({}, () => false), null);
});

test("specSkillReminder fires for brainstorming and writing-plans only", () => {
  assert.match(specSkillReminder({ tool_input: { skill: "superpowers:brainstorming" } }), /skill/i);
  assert.match(specSkillReminder({ tool_input: { skill: "writing-plans" } }), /skill/i);
  assert.equal(specSkillReminder({ tool_input: { skill: "tdd" } }), null);
  assert.equal(specSkillReminder({}), null);
});
