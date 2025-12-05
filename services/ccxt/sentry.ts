import type { MiddlewareHandler } from "hono";
import * as Sentry from "@sentry/bun";

const sentryDsn = process.env.SENTRY_DSN;
const sentryEnvironment =
  process.env.SENTRY_ENVIRONMENT || process.env.NODE_ENV || "development";
const sentryRelease = process.env.SENTRY_RELEASE;

const tracesSampleRate = Number(
  process.env.SENTRY_TRACES_SAMPLE_RATE || "0.2",
);
const profilesSampleRate = Number(
  process.env.SENTRY_PROFILES_SAMPLE_RATE || "0",
);

if (sentryDsn) {
  Sentry.init({
    dsn: sentryDsn,
    environment: sentryEnvironment,
    release: sentryRelease,
    tracesSampleRate: Number.isFinite(tracesSampleRate)
      ? tracesSampleRate
      : 0,
    // @ts-ignore - profilesSampleRate is not present in @sentry/bun BunOptions type definition
    // as of version 7.90.0. This field enables profiling for performance monitoring.
    // See: https://docs.sentry.io/platforms/javascript/profiling/
    profilesSampleRate: Number.isFinite(profilesSampleRate)
      ? profilesSampleRate
      : 0,
    attachStacktrace: true,
  });
}

export const isSentryEnabled = !!sentryDsn;

export const sentryMiddleware: MiddlewareHandler = async (c, next) => {
  if (!isSentryEnabled) {
    return next();
  }

  return Sentry.startSpan(
    {
      name: `${c.req.method} ${c.req.path}`,
      op: "http.server",
      attributes: {
        request_path: c.req.path,
      },
    },
    async (span) => {
      try {
        await next();
        span.setAttribute("http.response.status_code", c.res.status);
        span.setAttribute("request.host", c.req.header("host") || "unknown");
      } catch (error) {
        Sentry.captureException(error);
        span.setStatus({ code: 2 }); // ERROR
        throw error;
      }
    }
  );
};
