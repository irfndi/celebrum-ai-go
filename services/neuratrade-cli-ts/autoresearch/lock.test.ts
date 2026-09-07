import { describe, expect, it } from "bun:test";
import { mkdtempSync, rmSync, writeFileSync, existsSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { withFileLock } from "./lock.ts";

describe("withFileLock", () => {
  it("recovers an abandoned lock", () => {
    const dir = mkdtempSync(join(tmpdir(), "ar-lock-stale-"));
    const lock = join(dir, "x.lock");
    try {
      // Simulate a crashed worker: stale owner payload, never released.
      writeFileSync(
        lock,
        JSON.stringify({
          pid: 999999,
          acquiredAt: Date.now() - 60_000,
          owner: "dead:1:abc",
        }),
      );
      let runs = 0;
      const out = withFileLock(
        lock,
        () => {
          runs += 1;
          return "ok";
        },
        { retries: 50, sleepMs: 5, staleMs: 1_000 },
      );
      expect(out).toBe("ok");
      expect(runs).toBe(1);
      expect(existsSync(lock)).toBe(false);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });

  it("runs a failing callback once and throws the original error", () => {
    const dir = mkdtempSync(join(tmpdir(), "ar-lock-err-"));
    const lock = join(dir, "x.lock");
    try {
      let runs = 0;
      const original = new Error("boom-original");
      let caught: unknown;
      try {
        withFileLock(
          lock,
          () => {
            runs += 1;
            throw original;
          },
          { retries: 5, sleepMs: 1 },
        );
      } catch (e) {
        caught = e;
      }
      expect(caught).toBe(original);
      expect(runs).toBe(1);
      // Lock must be released even on callback failure.
      expect(existsSync(lock)).toBe(false);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });
});
