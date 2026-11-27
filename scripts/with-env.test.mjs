import assert from "node:assert/strict";
import { test } from "node:test";

import { mergeEnv, missing, parseArgs } from "./with-env.mjs";

test("mergeEnv fills in what the environment lacks from the .env file", () => {
  const merged = mergeEnv('DATABASE_URL=postgres://localhost/horizon\nSMTP_FROM="Horizon Money <noreply@example.com>"\n', {
    PATH: "/usr/bin",
  });
  assert.equal(merged.DATABASE_URL, "postgres://localhost/horizon");
  assert.equal(merged.SMTP_FROM, "Horizon Money <noreply@example.com>");
  assert.equal(merged.PATH, "/usr/bin");
});

test("mergeEnv never overrides a variable the environment already sets", () => {
  const merged = mergeEnv("PORT=8080\n", { PORT: "9090" });
  assert.equal(merged.PORT, "9090");
});

test("mergeEnv without a .env file keeps the environment as is", () => {
  assert.deepEqual(mergeEnv(null, { PORT: "9090" }), { PORT: "9090" });
});

test("parseArgs splits the --require variables from the command", () => {
  assert.deepEqual(parseArgs(["--require", "TEST_DATABASE_URL", "go", "test", "./..."]), {
    required: ["TEST_DATABASE_URL"],
    command: "go",
    args: ["test", "./..."],
  });
  assert.deepEqual(parseArgs(["go", "run", "."]), { required: [], command: "go", args: ["run", "."] });
});

test("missing lists the required variables that are unset or empty", () => {
  assert.deepEqual(missing(["A", "B", "C"], { A: "1", B: "" }), ["B", "C"]);
});
