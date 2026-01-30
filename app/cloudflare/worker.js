const CLOUD_RUN_URL = "https://prophet-agent-594677951902.us-west1.run.app";

export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    const targetUrl = CLOUD_RUN_URL + url.pathname + url.search;
    
    const newRequest = new Request(targetUrl, {
      method: request.method,
      headers: request.headers,
      body: request.body,
      redirect: "manual"
    });
    
    const response = await fetch(newRequest);
    
    // Clone response with CORS headers
    const newResponse = new Response(response.body, {
      status: response.status,
      statusText: response.statusText,
      headers: response.headers
    });
    
    return newResponse;
  }
}
