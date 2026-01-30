/**
 * Temple Square Cloudflare Worker
 * Proxies requests from templesquare.dev to Cloud Run backend
 * Handles SSE streaming for real-time responses
 */

const BASIC_AUTH_REALM = "Temple Square";

function timingSafeEqual(a, b) {
  const left = typeof a === "string" ? a : "";
  const right = typeof b === "string" ? b : "";
  const maxLen = Math.max(left.length, right.length);
  let result = 0;
  for (let i = 0; i < maxLen; i += 1) {
    const leftCode = i < left.length ? left.charCodeAt(i) : 0;
    const rightCode = i < right.length ? right.charCodeAt(i) : 0;
    result |= leftCode ^ rightCode;
  }
  return result === 0 && left.length === right.length;
}

function isAuthorized(request, env) {
  const expectedPassword = env.BASIC_AUTH_PASSWORD;
  if (!expectedPassword) {
    return true; // auth disabled if no password is configured
  }

  const authorization = request.headers.get("Authorization") || "";
  if (!authorization.startsWith("Basic ")) {
    return false;
  }

  let decoded = "";
  try {
    decoded = atob(authorization.slice("Basic ".length));
  } catch {
    return false;
  }

  const separatorIndex = decoded.indexOf(":");
  const providedUser =
    separatorIndex >= 0 ? decoded.slice(0, separatorIndex) : "";
  const providedPassword =
    separatorIndex >= 0 ? decoded.slice(separatorIndex + 1) : "";

  if (
    env.BASIC_AUTH_USERNAME &&
    !timingSafeEqual(providedUser, env.BASIC_AUTH_USERNAME)
  ) {
    return false;
  }

  return timingSafeEqual(providedPassword, expectedPassword);
}

function unauthorizedResponse() {
  return new Response("Unauthorized", {
    status: 401,
    headers: {
      "WWW-Authenticate": `Basic realm="${BASIC_AUTH_REALM}"`,
      "Cache-Control": "no-store",
    },
  });
}

export default {
  async fetch(request, env, ctx) {
    if (!isAuthorized(request, env)) {
      return unauthorizedResponse();
    }

    const url = new URL(request.url);

    // Build the backend URL
    const backendUrl = new URL(url.pathname + url.search, env.BACKEND_URL);

    const forwardHeaders = new Headers(request.headers);
    if (env.BASIC_AUTH_PASSWORD) {
      forwardHeaders.delete("authorization");
    }

    // Clone the request with the new URL
    const backendRequest = new Request(backendUrl, {
      method: request.method,
      headers: forwardHeaders,
      body: request.body,
      redirect: 'follow',
    });

    // Forward the request to Cloud Run
    const response = await fetch(backendRequest);

    // Check if this is an SSE response
    const contentType = response.headers.get('content-type') || '';

    if (contentType.includes('text/event-stream')) {
      // For SSE, we need to stream the response
      // Create new headers without content-encoding to prevent buffering
      const headers = new Headers(response.headers);
      headers.delete('content-encoding');
      headers.set('cache-control', 'no-cache');
      headers.set('connection', 'keep-alive');

      return new Response(response.body, {
        status: response.status,
        statusText: response.statusText,
        headers: headers,
      });
    }

    // For non-SSE responses, just pass through
    return response;
  },
};
