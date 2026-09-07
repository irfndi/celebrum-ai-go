import { describe, expect, it } from "bun:test";

/**
 * P2 clever-cabin-x0i: lodash via alchemy > @prisma/dev > @mrleebo/prisma-ast >
 * chevrotain > @chevrotain/gast carried GHSA-r5fr-rjxr-66jc (high) + two
 * moderate prototype-pollution advisories. Upstream chevrotain pins
 * lodash@4.17.21 exactly, so `bun audit fix` is blocked by the dependent
 * range — the override below forces the fixed 4.18.x line.
 */
describe("CLI dependency audit guard (clever-cabin-x0i)", () => {
  it("pins lodash past the vulnerable range via overrides", async () => {
    const pkg = (await Bun.file(
      new URL("./package.json", import.meta.url),
    ).json()) as { overrides?: Record<string, string> };
    const pinned = pkg.overrides?.["lodash"];
    expect(pinned).toBeDefined();
    // Vulnerable range is <=4.17.23; require the fixed 4.18.x line.
    expect(pinned ?? "").toMatch(/4\.18/);
  });

  it("resolves lodash to a non-vulnerable version in bun.lock", async () => {
    const lock = await Bun.file(new URL("./bun.lock", import.meta.url)).text();
    const m = lock.match(/"lodash@([^"]+)"/);
    expect(m).not.toBeNull();
    const version = m![1];
    const parts = version.split(".").map(Number);
    const vulnerable =
      parts[0] < 4 ||
      (parts[0] === 4 && parts[1] < 18) ||
      (parts[0] === 4 && parts[1] === 17 && parts[2] <= 23);
    expect(vulnerable).toBe(false);
  });
});
