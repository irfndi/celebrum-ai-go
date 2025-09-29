/**
 * Normalize an OTLP traces endpoint to a URL that ends with `/v1/traces`.
 *
 * If `url` is falsy, returns `null`. If `url` already specifies an HTTP(S) origin
 * or a host (missing scheme), the result is a URL string whose path ends with
 * `/v1/traces`. For malformed inputs the function returns the original value
 * with any trailing slash removed and `/v1/traces` appended.
 *
 * @param {string|null|undefined} url - The endpoint to normalize; may be a full URL, a host without scheme, or a malformed value.
 * @returns {string|null} A normalized URL string ending with `/v1/traces`, or `null` if `url` is falsy.
 */
function ensureTracesPath(url) {
  if (!url) return null;
  try {
    const u = new URL(url);
    // Check if it's a proper URL with http/https scheme
    if (u.protocol === "http:" || u.protocol === "https:") {
      if (/\/v1\/traces\/?$/.test(u.pathname)) return u.toString();
      u.pathname = `${u.pathname.replace(/\/$/, "")}/v1/traces`;
      return u.toString();
    } else {
      // Not a proper http/https URL, use fallback
      throw new Error("Not a proper URL");
    }
  // eslint-disable-next-line no-unused-vars
  } catch (e) {
    // Fallback when URL lacks scheme (e.g., collector:4318)
    // Add http:// scheme to make it parseable by URL constructor
    const urlWithScheme = `http://${url}`;
    try {
      const u = new URL(urlWithScheme);
      if (/\/v1\/traces\/?$/.test(u.pathname)) return u.toString();
      u.pathname = `${u.pathname.replace(/\/$/, "")}/v1/traces`;
      return u.toString();
    // eslint-disable-next-line no-unused-vars
    } catch (e2) {
      // Final fallback for malformed URLs
      if (/\/v1\/traces\/?$/.test(url)) return url;
      return `${String(url).replace(/\/$/, "")}/v1/traces`;
    }
  }
}

<<<<<<< HEAD
function resolveOtlpTracesEndpoint(
  otlpTracesEnv,
  otlpBaseEnv,
  defaultBaseUrl = "http://localhost:4318",
) {
=======
/**
 * Selects the OTLP traces endpoint using environment-provided values with normalized fallbacks.
 *
 * @param {string|undefined|null} otlpTracesEnv - Explicit OTLP traces endpoint (takes highest precedence) as provided by an environment variable.
 * @param {string|undefined|null} otlpBaseEnv - Base OTLP URL (used if `otlpTracesEnv` is falsy); will be normalized to include the `/v1/traces` path.
 * @param {string} [defaultBaseUrl='http://localhost:4318'] - Fallback base URL used when the other inputs are falsy; will be normalized to include the `/v1/traces` path.
 * @returns {string|null} The chosen OTLP traces endpoint URL, or `null` if all inputs are falsy.
 */
function resolveOtlpTracesEndpoint(otlpTracesEnv, otlpBaseEnv, defaultBaseUrl = 'http://localhost:4318') {
>>>>>>> 47dcf0701519045878635a277800880c3ffecca4
  return (
    otlpTracesEnv ||
    ensureTracesPath(otlpBaseEnv) ||
    ensureTracesPath(defaultBaseUrl)
  );
}

module.exports = { ensureTracesPath, resolveOtlpTracesEndpoint };
