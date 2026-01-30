/**
 * Temple Square Cloudflare Worker
 * Proxies requests from templesquare.dev to Cloud Run backend
 * Handles SSE streaming for real-time responses
 */

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);

    // Build the backend URL
    const backendUrl = new URL(url.pathname + url.search, env.BACKEND_URL);

    // Clone the request with the new URL
    const backendRequest = new Request(backendUrl, {
      method: request.method,
      headers: request.headers,
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
