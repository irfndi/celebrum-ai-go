// Utility helpers for OTLP traces endpoint normalization
function ensureTracesPath(url) {
  if (!url) return null;
  try {
    const u = new URL(url);
    // Check if it's a proper URL with http/https scheme
    if (u.protocol === 'http:' || u.protocol === 'https:') {
      if (/\/v1\/traces\/?$/.test(u.pathname)) return u.toString();
      u.pathname = `${u.pathname.replace(/\/$/, '')}/v1/traces`;
      return u.toString();
    } else {
      // Not a proper http/https URL, use fallback
      throw new Error('Not a proper URL');
    }
  } catch (e) {
    // Fallback when URL lacks scheme (e.g., collector:4318)
    if (/\/v1\/traces\/?$/.test(url)) return url;
    return `${String(url).replace(/\/$/, '')}/v1/traces`;
  }
}

function resolveOtlpTracesEndpoint(otlpTracesEnv, otlpBaseEnv, defaultBaseUrl = 'http://localhost:4318') {
  return (
    otlpTracesEnv ||
    ensureTracesPath(otlpBaseEnv) ||
    ensureTracesPath(defaultBaseUrl)
  );
}

module.exports = { ensureTracesPath, resolveOtlpTracesEndpoint };