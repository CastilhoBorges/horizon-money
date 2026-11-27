#!/usr/bin/env node
// Creates and migrates one database per worktree on the shared compose Postgres (AGENTS.md rule 5), then
// prints the DATABASE_URL line for the worktree's .env.
//
// Usage: pnpm db:worktree <slug>      e.g. pnpm db:worktree persistence → horizon_persistence
import { execFileSync } from "node:child_process";
import { dirname, join } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const slugPattern = /^[a-z0-9]+(-[a-z0-9]+)*$/;

// databaseName maps a worktree slug to its database. Only lower-case letters, digits and single dashes
// pass, so the name is always a safe SQL identifier.
export function databaseName(slug) {
  if (!slugPattern.test(slug)) {
    throw new Error(`invalid slug ${JSON.stringify(slug)}: use lower-case letters, digits and dashes`);
  }
  return `horizon_${slug.replaceAll("-", "_")}`;
}

// databaseURL is the connection string for a database on the compose Postgres, honouring the same
// POSTGRES_* overrides compose.yaml reads.
export function databaseURL(name, env) {
  const user = encodeURIComponent(env.POSTGRES_USER || "horizon");
  const password = encodeURIComponent(env.POSTGRES_PASSWORD || "horizon");
  const port = env.POSTGRES_PORT || "5432";
  return `postgres://${user}:${password}@localhost:${port}/${name}?sslmode=disable`;
}

function main() {
  const root = join(dirname(fileURLToPath(import.meta.url)), "..");
  const name = databaseName(process.argv[2] ?? "");
  const user = process.env.POSTGRES_USER || "horizon";
  const psql = (sql) =>
    execFileSync("docker", ["compose", "exec", "-T", "postgres", "psql", "-U", user, "-d", "postgres", "-tAc", sql], {
      cwd: root,
      encoding: "utf8",
    }).trim();

  if (psql(`select 1 from pg_database where datname = '${name}'`) !== "1") {
    psql(`create database ${name}`);
  }
  const url = databaseURL(name, process.env);
  execFileSync("go", ["run", "./cmd/horizon-money", "migrate"], {
    cwd: join(root, "apps", "api"),
    env: { ...process.env, DATABASE_URL: url },
    stdio: ["ignore", "ignore", "inherit"],
  });
  console.log(`DATABASE_URL=${url}`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    main();
  } catch (err) {
    console.error(`db:worktree: ${err.message}`);
    process.exit(1);
  }
}
