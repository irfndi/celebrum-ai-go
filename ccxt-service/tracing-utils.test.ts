import { describe, expect, test } from 'bun:test';
// Use require to interop with CommonJS module
// eslint-disable-next-line @typescript-eslint/no-var-requires
const { ensureTracesPath, resolveOtlpTracesEndpoint } = require('./tracing-utils.js');

describe('tracing-utils ensureTracesPath', () => {
  test('appends /v1/traces to base URL with no path', () => {
    const out = ensureTracesPath('http://localhost:4318');
    expect(new URL(out).pathname).toBe('/v1/traces');
  });

  test('normalizes trailing slash on base URL', () => {
    const out = ensureTracesPath('http://collector:4318/');
    expect(new URL(out).pathname).toBe('/v1/traces');
  });

  test('keeps existing /v1/traces path', () => {
    const out = ensureTracesPath('http://collector:4318/v1/traces');
    expect(new URL(out).pathname).toBe('/v1/traces');
  });

  test('handles scheme-less host:port', () => {
    const out = ensureTracesPath('collector:4318');
    expect(new URL(out).pathname).toBe('/v1/traces');
  });

  test('appends to custom base path', () => {
    const out = ensureTracesPath('https://otlp.example.com:4318/otlp');
    expect(new URL(out).pathname).toBe('/otlp/v1/traces');
  });
});

describe('tracing-utils resolveOtlpTracesEndpoint', () => {
  test('prefers OTLP_TRACES env when provided', () => {
    const out = resolveOtlpTracesEndpoint('http://a:4318/v1/traces', 'http://b:4318');
    expect(out).toBe('http://a:4318/v1/traces');
  });

  test('falls back to normalized base endpoint', () => {
    const out = resolveOtlpTracesEndpoint(undefined, 'http://b:4318');
    expect(new URL(out).pathname).toBe('/v1/traces');
  });

  test('defaults to localhost base when nothing provided', () => {
    const out = resolveOtlpTracesEndpoint(undefined, undefined);
    expect(new URL(out).hostname).toBe('localhost');
    expect(new URL(out).pathname).toBe('/v1/traces');
  });
});