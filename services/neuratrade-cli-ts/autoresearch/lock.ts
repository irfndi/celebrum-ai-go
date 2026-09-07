/**
 * Exclusive file lock for shared champion.json across parallel workers.
 *
 * Each lock file carries owner identity (pid + timestamp + token) so an
 * abandoned lock (owner crashed without releasing) can be recovered after
 * a TTL. Only lock-acquisition failures are retried; callback errors are
 * never retried and the original error propagates with the callback
 * executed exactly once.
 */
import {
  openSync,
  closeSync,
  unlinkSync,
  writeFileSync,
  readFileSync,
  mkdirSync,
  statSync,
  writeSync,
} from "node:fs";
import { dirname } from "node:path";

export interface LockOptions {
  retries?: number;
  sleepMs?: number;
  /** TTL after which an unreleased lock is treated as abandoned. Defaults to 30s. */
  staleMs?: number;
}

interface LockPayload {
  pid: number;
  acquiredAt: number;
  owner: string;
}

function readPayload(lockPath: string): LockPayload | null {
  try {
    return JSON.parse(readFileSync(lockPath, "utf8")) as LockPayload;
  } catch {
    return null;
  }
}

function isStale(lockPath: string, staleMs: number, now: number): boolean {
  const payload = readPayload(lockPath);
  if (payload && Number.isFinite(payload.acquiredAt)) {
    return now - payload.acquiredAt >= staleMs;
  }
  // Unparseable lock file: fall back to mtime so a corrupt/legacy
  // lock cannot block workers forever.
  try {
    const mtime = statSync(lockPath).mtimeMs;
    return now - mtime >= staleMs;
  } catch {
    // File vanished between attempts: not stale, just retry acquisition.
    return false;
  }
}

function tryRecoverStaleLock(lockPath: string, staleMs: number): void {
  try {
    if (isStale(lockPath, staleMs, Date.now())) {
      unlinkSync(lockPath);
    }
  } catch {
    /* ignore: missing file or raced unlink */
  }
}

export function withFileLock<T>(
  lockPath: string,
  fn: () => T,
  opts: LockOptions = {},
): T {
  const retries = opts.retries ?? 200;
  const sleepMs = opts.sleepMs ?? 25;
  const staleMs = opts.staleMs ?? 30_000;
  mkdirSync(dirname(lockPath), { recursive: true });

  const owner = `${process.pid}:${Date.now()}:${Math.random().toString(36).slice(2)}`;
  const payload: LockPayload = {
    pid: process.pid,
    acquiredAt: Date.now(),
    owner,
  };
  acquireLock(lockPath, payload, retries, sleepMs, staleMs);

  // Run the critical section exactly once. Callback errors propagate
  // untouched and are never retried.
  try {
    return fn();
  } finally {
    // Release only our own lock so we never delete a fresh lock that a
    // racing worker (or a stale-recovery) installed.
    releaseOwnLock(lockPath, owner);
  }
}

function closeIgnoringErrors(fd: number): void {
  try {
    closeSync(fd);
  } catch {
    /* ignore */
  }
}

function unlinkIgnoringErrors(path: string): void {
  try {
    unlinkSync(path);
  } catch {
    /* ignore */
  }
}

/**
 * One acquisition attempt: exclusive-create the lock file and stamp our
 * payload. Returns the open fd on success, null when another worker holds
 * the lock or our own stamp failed (lock file removed).
 */
function acquireLockOnce(
  lockPath: string,
  payload: LockPayload,
): number | null {
  let fd: number;
  try {
    fd = openSync(lockPath, "wx");
  } catch {
    return null;
  }
  try {
    writeSync(fd, JSON.stringify(payload));
    return fd;
  } catch {
    closeIgnoringErrors(fd);
    unlinkIgnoringErrors(lockPath);
    return null;
  }
}

function acquireLock(
  lockPath: string,
  payload: LockPayload,
  retries: number,
  sleepMs: number,
  staleMs: number,
): void {
  for (let i = 0; i < retries; i++) {
    const fd = acquireLockOnce(lockPath, payload);
    if (fd !== null) {
      closeIgnoringErrors(fd);
      return;
    }
    // Acquisition failure only: maybe the holder crashed. Recover
    // stale locks, then wait and retry.
    tryRecoverStaleLock(lockPath, staleMs);
    Bun.sleepSync(sleepMs);
  }
  throw new Error(`timeout acquiring lock ${lockPath}`);
}

function releaseOwnLock(lockPath: string, owner: string): void {
  try {
    const current = readPayload(lockPath);
    if (current?.owner === owner) {
      unlinkSync(lockPath);
    }
  } catch {
    /* ignore */
  }
}

export function readJsonFile<T>(path: string): T | null {
  try {
    return JSON.parse(readFileSync(path, "utf8")) as T;
  } catch {
    return null;
  }
}

export function writeJsonFile<T>(path: string, value: T): void {
  writeFileSync(path, `${JSON.stringify(value, null, 2)}\n`);
}
