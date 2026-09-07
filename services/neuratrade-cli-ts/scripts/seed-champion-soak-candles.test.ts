import { describe, expect, test } from "bun:test";
import { Database } from "bun:sqlite";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const SEED_PY = readFileSync(
  join(import.meta.dir, "seed-champion-soak-candles.py"),
  "utf8",
);

describe("seed-champion-soak-candles kill-switch (P0)", () => {
  test("incremental sync preserves the switch, only creating when missing", () => {
    // The old code ran engaged=0 on every --incremental run, clearing a live stop.
    expect(SEED_PY).toContain("ON CONFLICT(id) DO NOTHING");
    // Incremental kill-switch branch must not reset engaged.
    const ksIdx = SEED_PY.indexOf("P0: incremental sync must never clear");
    expect(ksIdx).toBeGreaterThan(-1);
    const incrKsIdx = SEED_PY.lastIndexOf("if incremental:", ksIdx);
    expect(incrKsIdx).toBeGreaterThan(-1);
    const elseAfter = SEED_PY.indexOf("\n    else:", ksIdx);
    expect(elseAfter).toBeGreaterThan(ksIdx);
    const incrOnly = SEED_PY.slice(incrKsIdx, elseAfter);
    expect(incrOnly).toContain("DO NOTHING");
    expect(incrOnly).not.toContain("engaged = 0");
    // Full-seed else branch still resets for first boot.
    const elseBlock = SEED_PY.slice(
      elseAfter,
      SEED_PY.indexOf("dst.commit()", elseAfter),
    );
    expect(elseBlock).toContain("engaged = 0");
  });

  test("engaged switch survives the incremental SQL; full seed resets", () => {
    const db = new Database(":memory:");
    db.exec(`CREATE TABLE risk_kill_switch (
      id INTEGER PRIMARY KEY CHECK (id = 1),
      engaged BOOLEAN NOT NULL DEFAULT 0,
      reason TEXT NOT NULL DEFAULT '',
      updated_at DATETIME NOT NULL
    )`);
    db.exec(
      `INSERT INTO risk_kill_switch (id, engaged, reason, updated_at) VALUES (1, 1, 'manual stop', datetime('now'))`,
    );
    // Incremental path from the fixed script.
    db.exec(
      `INSERT INTO risk_kill_switch (id, engaged, reason, updated_at) VALUES (1, 0, '', datetime('now')) ON CONFLICT(id) DO NOTHING`,
    );
    const kept = db
      .query("SELECT engaged, reason FROM risk_kill_switch WHERE id = 1")
      .get() as any;
    expect(Number(kept.engaged)).toBe(1);
    expect(kept.reason).toBe("manual stop");

    // Missing row is still created disengaged.
    db.exec("DELETE FROM risk_kill_switch WHERE id = 1");
    db.exec(
      `INSERT INTO risk_kill_switch (id, engaged, reason, updated_at) VALUES (1, 0, '', datetime('now')) ON CONFLICT(id) DO NOTHING`,
    );
    const created = db
      .query("SELECT engaged FROM risk_kill_switch WHERE id = 1")
      .get() as any;
    expect(Number(created.engaged)).toBe(0);
    db.close();
  });
});

describe("seed-champion-soak-candles watermarks + open candle", () => {
  test("watermark is per (pair, timeframe), not per timeframe", () => {
    expect(SEED_PY).toContain("GROUP BY trading_pair_id, timeframe");
    expect(SEED_PY).toContain("trading_pair_id = ? AND timeframe = ?");

    // Behavioral proof: two symbols with different tips keep both watermarks.
    const db = new Database(":memory:");
    db.exec(`CREATE TABLE ohlcv_data (
      exchange_id INTEGER NOT NULL,
      trading_pair_id INTEGER NOT NULL,
      timeframe TEXT NOT NULL,
      timestamp DATETIME NOT NULL
    )`);
    const ins = db.prepare(
      "INSERT INTO ohlcv_data (exchange_id, trading_pair_id, timeframe, timestamp) VALUES (?, ?, ?, ?)",
    );
    ins.run(1, 10, "15m", "2026-08-13T12:00:00.000Z"); // BTC ahead
    ins.run(1, 11, "15m", "2026-08-13T11:00:00.000Z"); // LINK stale
    const rows = db
      .query(
        "SELECT trading_pair_id, timeframe, MAX(timestamp) FROM ohlcv_data WHERE exchange_id = ? GROUP BY trading_pair_id, timeframe",
      )
      .all(1) as any[];
    expect(rows.length).toBe(2);
    // Old per-timeframe GROUP BY would collapse to 1 row and hide LINK's gap.
    const collapsed = db
      .query(
        "SELECT timeframe, MAX(timestamp) FROM ohlcv_data WHERE exchange_id = ? GROUP BY timeframe",
      )
      .all(1) as any[];
    expect(collapsed.length).toBe(1);
    db.close();
  });

  test("open candle is excluded via per-timeframe cutoff", () => {
    expect(SEED_PY).toContain("open_candle_cutoff_iso");
    expect(SEED_PY).toContain("timestamp < ?");

    // Execute the real python helper to prove the cutoff floors correctly.
    const proc = Bun.spawnSync([
      "python3",
      "-c",
      "import importlib.util; spec = importlib.util.spec_from_file_location('seed', 'scripts/seed-champion-soak-candles.py'); m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m); from datetime import datetime, timezone; print(m.open_candle_cutoff_iso('15m', datetime(2026, 8, 13, 12, 7, 30, tzinfo=timezone.utc))); print(m.open_candle_cutoff_iso('5m', datetime(2026, 8, 13, 12, 7, 30, tzinfo=timezone.utc)))",
    ]);
    const out = proc.stdout.toString().trim().split("\n");
    expect(out[0]).toBe("2026-08-13T12:00:00.000Z");
    expect(out[1]).toBe("2026-08-13T12:05:00.000Z");
  });
});
