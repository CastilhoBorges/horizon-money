#!/usr/bin/env node
// Generates the docs indexes and THIRD_PARTY_NOTICES.md, and enforces the living-docs line limit.
//
//   node scripts/docs-index.mjs          write indexes and notices, then check line limits
//   node scripts/docs-index.mjs --check  fail if anything is stale or oversize (CI)
//
// See docs/README.md and ADR-0008 for the rules this script enforces.

import { existsSync, mkdirSync, readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

export const LINE_LIMIT = 300;

const START = "<!-- index:start -->";
const END = "<!-- index:end -->";

// License of every source repository that skills are vendored from. A new source must be added here,
// which forces someone to check its license before it lands.
export const SKILL_SOURCE_LICENSES = {
  "addyosmani/agent-skills": "MIT",
  "anthropics/skills": "Apache-2.0",
  "docker/skills": "Apache-2.0",
  "github/awesome-copilot": "MIT",
  "mattpocock/skills": "MIT",
  "obra/superpowers": "MIT",
  "samber/cc-skills-golang": "MIT",
  "supabase/agent-skills": "MIT",
  "vercel-labs/skills": "MIT",
  "vercel/turborepo": "MIT",
};

export function parseAdr(file, text) {
  const lines = text.split("\n");
  const heading = lines.find((line) => line.startsWith("# ")) ?? "";
  const match = heading.match(/^# ADR-(\d{4}):\s*(.+)$/);
  const statusAt = lines.findIndex((line) => line.trim() === "## Status");
  const status = statusAt >= 0 ? (lines.slice(statusAt + 1).find((l) => l.trim() !== "") ?? "").trim() : "";
  return {
    file,
    number: match ? match[1] : file.slice(0, 4),
    title: match ? match[2].trim() : heading.replace(/^#\s*/, ""),
    status,
  };
}

export function replaceIndex(readme, content) {
  const start = readme.indexOf(START);
  const end = readme.indexOf(END);
  if (start === -1 || end === -1 || end < start) {
    return `${readme.replace(/\n*$/, "")}\n\n${START}\n${content}\n${END}\n`;
  }
  return `${readme.slice(0, start)}${START}\n${content}\n${readme.slice(end)}`;
}

export function renderNotices(lock, licenses) {
  const bySource = new Map();
  for (const [name, entry] of Object.entries(lock.skills ?? {})) {
    const license = licenses[entry.source];
    if (!license) {
      throw new Error(
        `skills-lock.json has skills from ${entry.source}, whose license is unknown. ` +
          "Check it and add it to SKILL_SOURCE_LICENSES in scripts/docs-index.mjs.",
      );
    }
    const row = bySource.get(entry.source) ?? { license, commit: entry.commit, skills: [] };
    row.skills.push(name);
    bySource.set(entry.source, row);
  }

  const rows = [...bySource.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([source, row]) => {
      const skills = row.skills.sort().join(", ");
      return `| [${source}](https://github.com/${source}) | ${row.license} | ${skills} | \`${row.commit.slice(0, 7)}\` |`;
    });

  return [
    "# Third-party notices",
    "",
    "Horizon Money is MIT licensed. The agent skills in `.agents/skills/` are vendored from the projects",
    "below and keep their own licenses. Each skill folder holds the files as published at the listed commit",
    "(a skill that ships a LICENSE file keeps it). This file is generated from `skills-lock.json` by",
    "`pnpm docs:index`; do not edit it by hand.",
    "",
    "| Source | License | Skills | Commit |",
    "|---|---|---|---|",
    ...rows,
    "",
  ].join("\n");
}

export function findOversizeDocs(files, limit = LINE_LIMIT) {
  return Object.entries(files)
    .map(([path, text]) => ({ path, lines: text.replace(/\n$/, "").split("\n").length }))
    .filter(({ lines }) => lines > limit);
}

// ---------------------------------------------------------------------------------------------------
// Filesystem side

function listMarkdown(dir) {
  if (!existsSync(dir)) return [];
  return readdirSync(dir)
    .filter((name) => name.endsWith(".md") && name !== "README.md")
    .sort();
}

function walkMarkdown(dir) {
  if (!existsSync(dir)) return [];
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name);
    if (statSync(path).isDirectory()) return walkMarkdown(path);
    return name.endsWith(".md") && name !== "README.md" ? [path] : [];
  });
}

function firstHeading(text) {
  const line = text.split("\n").find((l) => l.startsWith("# "));
  return line ? line.slice(2).trim() : "";
}

function indexes(root) {
  const docs = join(root, "docs");

  const adrs = listMarkdown(join(docs, "decisions")).map((file) =>
    parseAdr(file, readFileSync(join(docs, "decisions", file), "utf8")),
  );
  const devlog = walkMarkdown(join(docs, "devlog"))
    .map((path) => ({ rel: relative(join(docs, "devlog"), path), name: path.split("/").pop() }))
    .sort((a, b) => b.name.localeCompare(a.name));
  const dated = (folder) =>
    listMarkdown(join(docs, folder)).map((file) => ({
      file,
      title: firstHeading(readFileSync(join(docs, folder, file), "utf8")),
    }));

  return [
    {
      path: join(docs, "decisions", "README.md"),
      heading: "# Architecture Decision Records\n\nOne file per decision, never rewritten. A changed decision gets a new ADR that supersedes the old one.",
      content: [
        "| ADR | Decision | Status |",
        "|---|---|---|",
        ...adrs.map((a) => `| [${a.number}](${a.file}) | ${a.title} | ${a.status} |`),
      ].join("\n"),
    },
    {
      path: join(docs, "devlog", "README.md"),
      heading: "# Devlog\n\nWhat was done and learned, newest first. One file per entry: `YYYY/YYYY-MM-DD-<slug>.md`.",
      content: devlog
        .map(({ rel, name }) => {
          const title = firstHeading(readFileSync(join(docs, "devlog", rel), "utf8"));
          return `- ${name.slice(0, 10)} — [${title}](${rel})`;
        })
        .join("\n") || "_No entries yet._",
    },
    {
      path: join(docs, "specs", "README.md"),
      heading: "# Specs\n\nApproved designs, one per feature: `YYYY-MM-DD-<topic>-design.md`.",
      content: dated("specs").map(({ file, title }) => `- ${file.slice(0, 10)} — [${title}](${file})`).join("\n") || "_No specs yet._",
    },
    {
      path: join(docs, "plans", "README.md"),
      heading: "# Plans\n\nImplementation plans, when a feature needs one: `YYYY-MM-DD-<topic>.md`.",
      content: dated("plans").map(({ file, title }) => `- ${file.slice(0, 10)} — [${title}](${file})`).join("\n") || "_No plans yet._",
    },
  ];
}

function livingDocs(root) {
  const docs = join(root, "docs");
  const paths = [
    join(root, "AGENTS.md"),
    join(docs, "README.md"),
    join(docs, "PROJECT.md"),
    join(docs, "GLOSSARY.md"),
    join(docs, "ROADMAP.md"),
    ...listMarkdown(join(docs, "architecture")).map((f) => join(docs, "architecture", f)),
    ...["architecture", "decisions", "devlog", "specs", "plans"].map((d) => join(docs, d, "README.md")),
  ];
  return Object.fromEntries(
    paths.filter((p) => existsSync(p)).map((p) => [relative(root, p), readFileSync(p, "utf8")]),
  );
}

function main() {
  const check = process.argv.includes("--check");
  const root = join(dirname(fileURLToPath(import.meta.url)), "..");
  const problems = [];

  const outputs = indexes(root).map(({ path, heading, content }) => {
    const current = existsSync(path) ? readFileSync(path, "utf8") : `${heading}\n`;
    return { path, current: existsSync(path) ? current : null, next: replaceIndex(current, content) };
  });

  const lock = JSON.parse(readFileSync(join(root, "skills-lock.json"), "utf8"));
  const noticesPath = join(root, "THIRD_PARTY_NOTICES.md");
  try {
    outputs.push({
      path: noticesPath,
      current: existsSync(noticesPath) ? readFileSync(noticesPath, "utf8") : null,
      next: renderNotices(lock, SKILL_SOURCE_LICENSES),
    });
  } catch (error) {
    problems.push(error.message);
  }

  for (const { path, current, next } of outputs) {
    if (current === next) continue;
    if (check) {
      problems.push(`${relative(root, path)} is stale. Run \`pnpm docs:index\` and commit the result.`);
    } else {
      mkdirSync(dirname(path), { recursive: true });
      writeFileSync(path, next);
      console.log(`updated ${relative(root, path)}`);
    }
  }

  for (const { path, lines } of findOversizeDocs(livingDocs(root))) {
    problems.push(`${path} has ${lines} lines; living docs stay under ${LINE_LIMIT}. Split it by area (ADR-0008).`);
  }

  if (problems.length > 0) {
    for (const problem of problems) console.error(`docs: ${problem}`);
    process.exit(1);
  }
  console.log(check ? "docs: indexes, notices and line limits are up to date" : "docs: done");
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main();
}
