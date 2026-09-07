import { describe, expect, it } from "bun:test";
import worker, {
  WHITELIST_MAX_AGE_MS,
  buildWatchlistEnvelope,
  isWhitelistStale,
  parseWhitelistMeta,
} from "./worker.ts";

// ---------------------------------------------------------------------------
// Universe-watch fetch handler routing. The /scan path hits live Bitget, so
// it is excluded; the admin-gated + KV-backed routes are covered with a fake
// env.
// ---------------------------------------------------------------------------

function fakeEnv(initial: Record<string, string> = {}) {
  const store = new Map(Object.entries(initial));
  return {
    watchlist: {
      get: async (k: string) => store.get(k) ?? null,
      put: async (k: string, v: string) => {
        store.set(k, v);
      },
    },
    adminKey: "test-admin-key",
  };
}

const req = (url: string, init?: RequestInit) =>
  worker.fetch(
    new Request(`https://universe-watch.example${url}`, init),
    fakeEnv() as never,
  );

const reqWith = (
  env: ReturnType<typeof fakeEnv>,
  url: string,
  init?: RequestInit,
) =>
  worker.fetch(
    new Request(`https://universe-watch.example${url}`, init),
    env as never,
  );

describe("universe-watch fetch handler", () => {
  it("serves /health unauthenticated", async () => {
    const res = await req("/health");
    expect(res.status).toBe(200);
    expect(((await res.json()) as { status: string }).status).toBe("healthy");
  });

  it("returns empty watchlist before any scan", async () => {
    const res = await req("/watchlist");
    expect(res.status).toBe(200);
    expect(await res.json()).toEqual({
      survivors: [],
      updatedAt: null,
      stale: true,
      entries: 0,
      note: "no scan run yet",
    });
  });

  it("rejects mutating routes without the admin key", async () => {
    const res = await req("/scan", { method: "POST" });
    expect(res.status).toBe(401);
  });

  it("rejects malformed JSON on /seed with a controlled 400", async () => {
    const res = await req("/seed", {
      method: "PUT",
      headers: {
        "x-api-key": "test-admin-key",
        "content-type": "application/json",
      },
      body: "{not json",
    });
    expect(res.status).toBe(400);
    expect(await res.json()).toEqual({ error: "Invalid JSON body" });
  });

  it("rejects empty symbol strings on /seed", async () => {
    const res = await req("/seed", {
      method: "PUT",
      headers: {
        "x-api-key": "test-admin-key",
        "content-type": "application/json",
      },
      body: JSON.stringify(["BTC/USDT", ""]),
    });
    expect(res.status).toBe(400);
  });

  it("stores a valid seed and reads it back", async () => {
    const mock = fakeEnv();
    const env = mock as never;
    const put = await worker.fetch(
      new Request("https://universe-watch.example/seed", {
        method: "PUT",
        headers: {
          "x-api-key": "test-admin-key",
          "content-type": "application/json",
        },
        body: JSON.stringify(["BTC/USDT", "ETH/USDT"]),
      }),
      env,
    );
    expect(put.status).toBe(200);
    expect(await mock.watchlist.get("seed-symbols")).toBe(
      '["BTC/USDT","ETH/USDT"]',
    );
  });
});

describe("whitelist freshness (clever-cabin-gzk)", () => {
  it("exposes freshness on GET /watchlist with a fresh meta", async () => {
    const updatedAt = new Date().toISOString();
    const env = fakeEnv({
      "grid-whitelist": JSON.stringify([{ symbol: "BTC/USDT" }]),
      "grid-whitelist-meta": JSON.stringify({
        updatedAt,
        entries: 10,
        survivors: 1,
        ok: true,
      }),
    });
    const res = await reqWith(env, "/watchlist");
    expect(res.status).toBe(200);
    const body = (await res.json()) as {
      survivors: unknown[];
      updatedAt: string;
      stale: boolean;
      entries: number;
    };
    expect(body.survivors).toHaveLength(1);
    expect(body.updatedAt).toBe(updatedAt);
    expect(body.stale).toBe(false);
    expect(body.entries).toBe(10);
  });

  it("marks a zero-survivor tombstone as fresh but empty", async () => {
    const updatedAt = new Date().toISOString();
    const env = fakeEnv({
      "grid-whitelist": "[]",
      "grid-whitelist-meta": JSON.stringify({
        updatedAt,
        entries: 10,
        survivors: 0,
        ok: true,
        note: "zero-survivors-tombstone",
      }),
    });
    const res = await reqWith(env, "/watchlist");
    const body = (await res.json()) as {
      survivors: unknown[];
      stale: boolean;
      note?: string;
    };
    expect(body.survivors).toEqual([]);
    expect(body.stale).toBe(false);
    expect(body.note).toBe("zero-survivors-tombstone");
  });

  it("fails closed on stale meta past the max age", async () => {
    const updatedAt = new Date(
      Date.now() - WHITELIST_MAX_AGE_MS - 1000,
    ).toISOString();
    const body = buildWatchlistEnvelope(
      JSON.stringify([{ symbol: "BTC/USDT" }]),
      JSON.stringify({ updatedAt, entries: 10, survivors: 1, ok: true }),
    );
    expect(body.stale).toBe(true);
    expect(body.updatedAt).toBe(updatedAt);
    // Former survivors are still returned for inspection, but flagged stale
    // so consumers must not treat them as currently eligible.
    expect(body.survivors).toHaveLength(1);
  });

  it("fails closed when the last scan errored (ok=false)", () => {
    expect(
      isWhitelistStale({
        updatedAt: new Date().toISOString(),
        entries: 10,
        survivors: 1,
        ok: false,
        error: "boom",
      }),
    ).toBe(true);
  });

  it("rejects invalid meta as stale", () => {
    expect(parseWhitelistMeta(null)).toBeNull();
    expect(parseWhitelistMeta("not-json")).toBeNull();
    expect(parseWhitelistMeta(JSON.stringify({ nope: 1 }))).toBeNull();
    expect(isWhitelistStale(null)).toBe(true);
  });

  it("serves /watchlist/meta before any scan", async () => {
    const res = await req("/watchlist/meta");
    expect(res.status).toBe(200);
    expect(await res.json()).toEqual({
      updatedAt: null,
      stale: true,
      note: "no scan run yet",
    });
  });
});
