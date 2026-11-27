#!/usr/bin/env node
// Claude Code hooks that inject short reminders of the AGENTS.md rules. They never block a tool call;
// the rules are the requirement, these are the backstop.
//
//   node .claude/hooks/reminders.mjs session     SessionStart
//   node .claude/hooks/reminders.mjs new-file    PreToolUse on Write
//   node .claude/hooks/reminders.mjs spec-skill  PreToolUse on Skill

import { existsSync, readFileSync } from "node:fs";
import { pathToFileURL } from "node:url";

const SOURCE_FILE = /\.(go|ts|tsx|mjs)$/;
const SPEC_SKILLS = /(^|:)(brainstorming|writing-plans)$/;

export function sessionContext() {
  return [
    "Horizon Money: AGENTS.md is the operating manual; docs/README.md is the map of what is true.",
    "Reuse before writing: search apps/ and packages/ for the thing you are about to add; the third copy is a defect (rule 11).",
    "Work in a worktree on a feature branch, test first, and close an issue with every PR.",
  ].join("\n");
}

export function newFileReminder(input, exists = existsSync) {
  const path = input?.tool_input?.file_path;
  if (typeof path !== "string" || !SOURCE_FILE.test(path) || exists(path)) return null;
  return (
    `You are creating a new source file (${path}). Rule 11, reuse before writing: search for an existing ` +
    "function, type or rule first (`rg -n \"<what it does>\" apps packages`). If this would be the third " +
    "copy of something, extract it into the layer that owns it instead."
  );
}

export function specSkillReminder(input) {
  const skill = input?.tool_input?.skill;
  if (typeof skill !== "string" || !SPEC_SKILLS.test(skill)) return null;
  return (
    "Rule 10, spec time is skill time: check the live skill list now, invoke every skill relevant to this " +
    "work, and list the skills you consulted in the spec."
  );
}

function emit(event, text) {
  if (!text) return;
  process.stdout.write(
    JSON.stringify({ hookSpecificOutput: { hookEventName: event, additionalContext: text } }),
  );
}

function readStdin() {
  try {
    const raw = readFileSync(0, "utf8");
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const mode = process.argv[2];
  if (mode === "session") emit("SessionStart", sessionContext());
  if (mode === "new-file") emit("PreToolUse", newFileReminder(readStdin()));
  if (mode === "spec-skill") emit("PreToolUse", specSkillReminder(readStdin()));
}
