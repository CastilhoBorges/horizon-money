#!/usr/bin/env node
// Runs a command with the repository's .env loaded, for the package scripts (Turborepo does not load
// .env files). Variables already in the environment always win over the file.
//
// Usage: node scripts/with-env.mjs [--require VAR]... <command> [args...]
// --require fails fast when VAR is still unset, instead of letting the command run without it.
import { spawn } from "node:child_process";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { parseEnv } from "node:util";

// mergeEnv returns env with the .env file's variables added where env lacks them. A null file means there
// is no .env.
export function mergeEnv(fileContent, env) {
  if (fileContent === null) {
    return { ...env };
  }
  return { ...parseEnv(fileContent), ...env };
}

// parseArgs splits the leading --require flags from the command and its arguments.
export function parseArgs(argv) {
  const required = [];
  let i = 0;
  while (argv[i] === "--require" && i + 1 < argv.length) {
    required.push(argv[i + 1]);
    i += 2;
  }
  return { required, command: argv[i], args: argv.slice(i + 1) };
}

// missing lists the required variables that env leaves unset or empty.
export function missing(required, env) {
  return required.filter((name) => !env[name]);
}

function readEnvFile(path) {
  try {
    return readFileSync(path, "utf8");
  } catch (err) {
    if (err.code === "ENOENT") {
      return null;
    }
    throw err;
  }
}

function main() {
  const { required, command, args } = parseArgs(process.argv.slice(2));
  if (!command) {
    console.error("with-env: usage: with-env.mjs [--require VAR]... <command> [args...]");
    process.exit(2);
  }
  const root = join(dirname(fileURLToPath(import.meta.url)), "..");
  const env = mergeEnv(readEnvFile(join(root, ".env")), process.env);
  const unset = missing(required, env);
  if (unset.length > 0) {
    console.error(`with-env: ${unset.join(", ")} not set; add it to .env (see .env.example)`);
    process.exit(1);
  }

  const child = spawn(command, args, { stdio: "inherit", env });
  for (const signal of ["SIGINT", "SIGTERM"]) {
    process.on(signal, () => child.kill(signal));
  }
  child.on("exit", (code, signal) => process.exit(code ?? (signal ? 1 : 0)));
  child.on("error", (err) => {
    console.error(`with-env: ${err.message}`);
    process.exit(1);
  });
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main();
}
