import { Effect, Layer, Option } from "effect";
import * as S from "effect/Schema";

import { MarketDataGateway } from "../market-data/gateway.js";
import { MarketDataGatewayLive } from "../market-data/gateways/index.js";
import {
  runGridUniverseScan,
  DEFAULT_GRID_UNIVERSE_SEARCH_SPACE,
  type GridUniverseOptions,
} from "../scalping/grid-universe.js";
import type { UniverseWatchEnv } from "../../alchemy.run.ts";
import { CloudflareMarketDataRepositoryLive } from "./market-data-repository.ts";

const EXCHANGE = "bitget-futures";
const TIMEFRAME = "15m";
const SEED_KEY = "seed-symbols";
const WATCHLIST_KEY = "grid-whitelist";

/**
 * P2 clever-cabin-gzk: a scan that finds zero survivors must expire the
 * previous whitelist instead of leaving it stale indefinitely.
 *
 * - WATCHLIST_META_KEY carries the freshness envelope for the whitelist:
 *   last successful scan timestamp, entry/survivor counts, and ok flag.
 * - Every successful scan writes BOTH keys, including the empty-array
 *   tombstone (`[]`) on zero survivors.
 * - GET /watchlist always returns the envelope `{ survivors, updatedAt,
 *   stale, ... }` so consumers can fail closed on stale data instead of
 *   treating former survivors as currently eligible.
 * - The scheduled handler rethrows scan errors (Cloudflare retry +
 *   observability) after a best-effort failure-status write — errors are
 *   surfaced, not swallowed.
 */
export const WATCHLIST_META_KEY = "grid-whitelist-meta";
/** Fail-closed freshness window: cron runs every 6h; >24h without a success is stale. */
export const WHITELIST_MAX_AGE_MS = 24 * 60 * 60 * 1000;

export interface WhitelistMeta {
  readonly updatedAt: string;
  readonly entries: number;
  readonly survivors: number;
  readonly ok: boolean;
  readonly note?: string;
  readonly error?: string;
}

const WhitelistMetaSchema = S.Struct({
  updatedAt: S.String,
  entries: S.optional(S.Number),
  survivors: S.Number,
  ok: S.Boolean,
  note: S.optional(S.String),
  error: S.optional(S.String),
});

export interface WatchlistEnvelope {
  readonly survivors: unknown[];
  readonly updatedAt: string | null;
  readonly stale: boolean;
  readonly entries: number;
  readonly note?: string;
}

export function parseWhitelistMeta(raw: string | null): WhitelistMeta | null {
  if (raw === null) return null;
  let decoded: unknown;
  try {
    decoded = JSON.parse(raw);
  } catch {
    return null;
  }
  const option = S.decodeUnknownOption(WhitelistMetaSchema)(decoded);
  if (Option.isNone(option)) return null;
  const value = option.value;
  const envelope: WhitelistMeta = {
    updatedAt: value.updatedAt,
    entries: value.entries ?? 0,
    survivors: value.survivors,
    ok: value.ok,
  };
  if (value.note !== undefined) {
    return { ...envelope, note: value.note };
  }
  if (value.error !== undefined) {
    return { ...envelope, error: value.error };
  }
  return envelope;
}

/** Fail closed: missing/invalid meta, ok=false, or older than the max age. */
export function isWhitelistStale(
  meta: WhitelistMeta | null,
  nowMs: number = Date.now(),
): boolean {
  if (meta === null || meta.ok === false) return true;
  const ts = Date.parse(meta.updatedAt);
  if (Number.isNaN(ts)) return true;
  return nowMs - ts > WHITELIST_MAX_AGE_MS;
}

export function buildWatchlistEnvelope(
  survivorsRaw: string | null,
  metaRaw: string | null,
  nowMs: number = Date.now(),
) {
  let survivors: unknown[] = [];
  if (survivorsRaw !== null) {
    try {
      const parsed: unknown = JSON.parse(survivorsRaw);
      survivors = Array.isArray(parsed) ? parsed : [];
    } catch {
      survivors = [];
    }
  }
  const meta = parseWhitelistMeta(metaRaw);
  if (meta === null) {
    if (survivorsRaw === null) {
      const fresh: WatchlistEnvelope = {
        survivors,
        updatedAt: null,
        stale: true,
        entries: 0,
        note: "no scan run yet",
      };
      return fresh;
    }
    const stale: WatchlistEnvelope = {
      survivors,
      updatedAt: null,
      stale: true,
      entries: 0,
    };
    return stale;
  }
  if (meta.note !== undefined) {
    const fresh: WatchlistEnvelope = {
      survivors,
      updatedAt: meta.updatedAt,
      stale: isWhitelistStale(meta, nowMs),
      entries: meta.entries,
      note: meta.note,
    };
    return fresh;
  }
  const plain: WatchlistEnvelope = {
    survivors,
    updatedAt: meta.updatedAt,
    stale: isWhitelistStale(meta, nowMs),
    entries: meta.entries,
  };
  return plain;
}

const DEFAULT_SEED_SYMBOLS = [
  "BTC/USDT",
  "ETH/USDT",
  "SOL/USDT",
  "ADA/USDT",
  "XRP/USDT",
  "HYPE/USDT",
  "HOME/USDT",
  "DOGE/USDT",
  "BNB/USDT",
  "LINK/USDT",
] as const;

function scanOptions(): GridUniverseOptions {
  return {
    exchange: EXCHANGE,
    timeframe: TIMEFRAME,
    initialCapital: 10000,
    // Live-fetch horizon: Bitget futures /history-candles caps at 200 per
    // request, so 500 is unreachable on the edge (SQLite local scans differ).
    minCandles: 200,
    trainWindow: 120,
    testWindow: 40,
    minProfitableWindowsPct: 60,
    minAggregateReturnPct: 0,
    feePct: 0.06,
    slippageBps: 2,
    trendFilterPeriod: 0,
    searchSpace: DEFAULT_GRID_UNIVERSE_SEARCH_SPACE,
  };
}

function readSeed(raw: string | null): string[] {
  if (raw === null) return [...DEFAULT_SEED_SYMBOLS];
  try {
    const parsed = JSON.parse(raw) as unknown;
    return Array.isArray(parsed)
      ? (parsed as string[])
      : [...DEFAULT_SEED_SYMBOLS];
  } catch {
    return [...DEFAULT_SEED_SYMBOLS];
  }
}

function runScan(kv: {
  get: (k: string) => Promise<string | null>;
  put: (k: string, v: string) => Promise<void>;
}) {
  return Effect.gen(function* () {
    const seed = yield* Effect.promise(() => kv.get(SEED_KEY)).pipe(
      Effect.map(readSeed),
      Effect.timeout("10 seconds"),
    );

    const gateway = yield* MarketDataGateway;

    const result = yield* runGridUniverseScan(scanOptions()).pipe(
      Effect.provide(
        Layer.provide(
          CloudflareMarketDataRepositoryLive(seed),
          Layer.succeed(MarketDataGateway, gateway),
        ),
      ),
    );

    // Same futures-symbol guard as the local scan: drop survivors whose
    // symbol has no Bitget USDT-M futures contract (public API, no creds).
    const futuresSymbols = yield* gateway
      .fetchSymbols(EXCHANGE)
      .pipe(Effect.catch(() => Effect.succeed([] as readonly string[])));
    const futuresSet = new Set(futuresSymbols);
    const canonical = (symbol: string) =>
      symbol.includes(":") ? symbol.slice(0, symbol.lastIndexOf(":")) : symbol;
    const survivors =
      futuresSet.size > 0
        ? result.survivors.filter(
            (e) =>
              futuresSet.has(e.symbol) || futuresSet.has(canonical(e.symbol)),
          )
        : result.survivors;

    const whitelist = survivors.map((e) => ({
      symbol: e.symbol,
      exchange: EXCHANGE,
      returnPct: e.walkForward.aggregateReturnPct,
      gridParams: {
        gridStepPct: e.bestParams.gridStepPct,
        gridMaxGrids: e.bestParams.gridMaxGrids,
        gridPauseAfterLossBars: e.bestParams.gridPauseAfterLossBars,
      },
    }));

    // Always persist — including the zero-survivor tombstone — so a fresh
    // empty scan expires the previous whitelist instead of leaving it stale.
    const base: WhitelistMeta = {
      updatedAt: new Date().toISOString(),
      entries: result.entries.length,
      survivors: whitelist.length,
      ok: true,
    };
    const meta: WhitelistMeta =
      whitelist.length === 0
        ? { ...base, note: "zero-survivors-tombstone" }
        : base;
    yield* Effect.promise(() =>
      kv.put(WATCHLIST_KEY, JSON.stringify(whitelist)),
    ).pipe(Effect.timeout("10 seconds"));
    yield* Effect.promise(() =>
      kv.put(WATCHLIST_META_KEY, JSON.stringify(meta)),
    ).pipe(Effect.timeout("10 seconds"));
    yield* Effect.log(
      `grid-universe scan: ${result.entries.length} symbols, ${result.survivors.length} survivors`,
    );
    return whitelist;
  }).pipe(Effect.provide(MarketDataGatewayLive));
}

function isAuthorizedRequest(request: Request, adminKey: string): boolean {
  return adminKey.length > 0 && request.headers.get("x-api-key") === adminKey;
}

function healthResponse(): Response {
  return Response.json({ status: "healthy", service: "universe-watch" });
}

function unauthorizedResponse(): Response {
  return Response.json({ error: "Unauthorized" }, { status: 401 });
}

function notFoundResponse(): Response {
  return new Response("Not Found", { status: 404 });
}

async function watchlistMetaResponse(env: UniverseWatchEnv): Promise<Response> {
  const metaRaw = await env.watchlist.get(WATCHLIST_META_KEY);
  const meta = parseWhitelistMeta(metaRaw);
  if (meta === null) {
    return Response.json(
      { updatedAt: null, stale: true, note: "no scan run yet" },
      { headers: { "cache-control": "no-store" } },
    );
  }
  return Response.json(
    { ...meta, stale: isWhitelistStale(meta) },
    { headers: { "cache-control": "no-store" } },
  );
}

async function watchlistResponse(env: UniverseWatchEnv): Promise<Response> {
  const [raw, metaRaw] = await Promise.all([
    env.watchlist.get(WATCHLIST_KEY),
    env.watchlist.get(WATCHLIST_META_KEY),
  ]);
  return Response.json(buildWatchlistEnvelope(raw, metaRaw), {
    headers: { "cache-control": "no-store" },
  });
}

async function scanResponse(env: UniverseWatchEnv): Promise<Response> {
  try {
    const survivors = await Effect.runPromise(runScan(env.watchlist));
    const metaRaw = await env.watchlist.get(WATCHLIST_META_KEY);
    const meta = parseWhitelistMeta(metaRaw);
    return Response.json({
      scanned: true,
      survivors,
      updatedAt: meta?.updatedAt ?? null,
      stale: false,
    });
  } catch (err) {
    return Response.json(
      { scanned: false, error: String(err) },
      { status: 500 },
    );
  }
}

function isSeedPayload(symbols: unknown): symbols is ReadonlyArray<string> {
  const decoded = S.decodeUnknownOption(S.Array(S.String))(symbols);
  return (
    Option.isSome(decoded) && !decoded.value.some((s) => s.trim().length === 0)
  );
}

async function seedResponse(
  request: Request,
  env: UniverseWatchEnv,
): Promise<Response> {
  let symbols: unknown;
  try {
    symbols = await request.json();
  } catch {
    return Response.json({ error: "Invalid JSON body" }, { status: 400 });
  }
  if (!isSeedPayload(symbols)) {
    return Response.json(
      { error: "seed must be a JSON array of symbol strings" },
      { status: 400 },
    );
  }
  await env.watchlist.put(SEED_KEY, JSON.stringify(symbols));
  return Response.json({ seed: symbols.length });
}

const PUBLIC_WATCHLIST_ROUTES: ReadonlyArray<{
  readonly suffix: string;
  readonly handle: (env: UniverseWatchEnv) => Promise<Response>;
}> = [
  { suffix: "/watchlist/meta", handle: watchlistMetaResponse },
  { suffix: "/watchlist", handle: watchlistResponse },
];

export default {
  async scheduled(
    _controller: { scheduledTime: number },
    env: UniverseWatchEnv,
  ): Promise<void> {
    try {
      await Effect.runPromise(runScan(env.watchlist));
    } catch (err) {
      console.error(`cron scan failed: ${String(err)}`);
      // Best-effort failure status without clobbering the last good
      // whitelist body; then rethrow so Cloudflare surfaces the error
      // (retry + logs) instead of swallowing it.
      try {
        const previous = parseWhitelistMeta(
          await env.watchlist.get(WATCHLIST_META_KEY),
        );
        const failure: WhitelistMeta = {
          updatedAt: previous?.updatedAt ?? new Date(0).toISOString(),
          entries: previous?.entries ?? 0,
          survivors: previous?.survivors ?? 0,
          ok: false,
          error: String(err).slice(0, 500),
        };
        await env.watchlist.put(WATCHLIST_META_KEY, JSON.stringify(failure));
      } catch {
        // Meta write is best-effort; the rethrow below is what matters.
      }
      throw err;
    }
  },

  async fetch(request: Request, env: UniverseWatchEnv): Promise<Response> {
    const url = new URL(request.url);
    if (request.method === "GET") {
      if (url.pathname.endsWith("/health")) return healthResponse();
      const route = PUBLIC_WATCHLIST_ROUTES.find((r) =>
        url.pathname.endsWith(r.suffix),
      );
      if (route !== undefined) return route.handle(env);
    }
    if (!isAuthorizedRequest(request, env.adminKey)) {
      return unauthorizedResponse();
    }
    if (request.method === "POST" && url.pathname.endsWith("/scan")) {
      return scanResponse(env);
    }
    if (request.method === "PUT" && url.pathname.endsWith("/seed")) {
      return seedResponse(request, env);
    }
    return notFoundResponse();
  },
};
