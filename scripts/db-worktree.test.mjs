import assert from "node:assert/strict";
import { test } from "node:test";

import { databaseName, databaseURL } from "./db-worktree.mjs";

test("databaseName maps a worktree slug to horizon_<slug> with underscores", () => {
  assert.equal(databaseName("persistence"), "horizon_persistence");
  assert.equal(databaseName("adr-postgres-19"), "horizon_adr_postgres_19");
});

test("databaseName refuses anything but lower-case letters, digits and dashes", () => {
  for (const slug of ["", "Feat", "feat_x", "feat x", "x';drop database horizon;--", "../x", "-"]) {
    assert.throws(() => databaseName(slug), /slug/, `slug ${JSON.stringify(slug)}`);
  }
});

test("databaseURL points at the compose Postgres with its defaults", () => {
  assert.equal(
    databaseURL("horizon_persistence", {}),
    "postgres://horizon:horizon@localhost:5432/horizon_persistence?sslmode=disable",
  );
});

test("databaseURL honours the compose overrides from the environment", () => {
  const env = { POSTGRES_USER: "ana", POSTGRES_PASSWORD: "p@ss", POSTGRES_PORT: "15432" };
  assert.equal(databaseURL("horizon_x", env), "postgres://ana:p%40ss@localhost:15432/horizon_x?sslmode=disable");
});
