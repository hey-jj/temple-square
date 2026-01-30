// cmd/server/main.go
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gofr.dev/pkg/gofr"
	gofrHTTP "gofr.dev/pkg/gofr/http"
	"gofr.dev/pkg/gofr/http/response"

	prophetagent "github.com/temple-square/prophet-agent/internal/agent"
	"github.com/temple-square/prophet-agent/internal/ui/components"
)

// assetsBaseURL is the base URL for static assets including headshots
var assetsBaseURL = getEnv("ASSETS_BASE_URL", "https://storage.googleapis.com/temple-square-assets")

// speakerHeadshots maps speaker names (and common variations) to their headshot slug.
// The full URL is constructed as: assetsBaseURL + "/headshots/" + slug + "-square.webp"
var speakerHeadshots = map[string]string{
	// First Presidency
	"Russell M. Nelson":           "russell-nelson",
	"President Russell M. Nelson": "russell-nelson",
	"President Nelson":            "russell-nelson",
	"Dallin H. Oaks":              "dallin-oaks",
	"President Dallin H. Oaks":    "dallin-oaks",
	"President Oaks":              "dallin-oaks",
	"Henry B. Eyring":             "henry-eyring",
	"President Henry B. Eyring":   "henry-eyring",
	"President Eyring":            "henry-eyring",

	// Quorum of the Twelve Apostles
	"Jeffrey R. Holland":           "jeffrey-holland",
	"Elder Jeffrey R. Holland":     "jeffrey-holland",
	"Elder Holland":                "jeffrey-holland",
	"Dieter F. Uchtdorf":           "dieter-uchtdorf",
	"Elder Dieter F. Uchtdorf":     "dieter-uchtdorf",
	"Elder Uchtdorf":               "dieter-uchtdorf",
	"David A. Bednar":              "david-bednar",
	"Elder David A. Bednar":        "david-bednar",
	"Elder Bednar":                 "david-bednar",
	"Quentin L. Cook":              "quentin-cook",
	"Elder Quentin L. Cook":        "quentin-cook",
	"Elder Cook":                   "quentin-cook",
	"D. Todd Christofferson":       "todd-christofferson",
	"Elder D. Todd Christofferson": "todd-christofferson",
	"Elder Christofferson":         "todd-christofferson",
	"Neil L. Andersen":             "neil-andersen",
	"Elder Neil L. Andersen":       "neil-andersen",
	"Elder Andersen":               "neil-andersen",
	"Ronald A. Rasband":            "ronald-rasband",
	"Elder Ronald A. Rasband":      "ronald-rasband",
	"Elder Rasband":                "ronald-rasband",
	"Gary E. Stevenson":            "gary-stevenson",
	"Elder Gary E. Stevenson":      "gary-stevenson",
	"Elder Stevenson":              "gary-stevenson",
	"Dale G. Renlund":              "dale-renlund",
	"Elder Dale G. Renlund":        "dale-renlund",
	"Elder Renlund":                "dale-renlund",
	"Gerrit W. Gong":               "gerrit-gong",
	"Elder Gerrit W. Gong":         "gerrit-gong",
	"Elder Gong":                   "gerrit-gong",
	"Ulisses Soares":               "ulisses-soares",
	"Elder Ulisses Soares":         "ulisses-soares",
	"Elder Soares":                 "ulisses-soares",
	"Patrick Kearon":               "patrick-kearon",
	"Elder Patrick Kearon":         "patrick-kearon",
	"Elder Kearon":                 "patrick-kearon",

	// Past Church Presidents (commonly quoted)
	"Gordon B. Hinckley":           "gordon-hinckley",
	"President Gordon B. Hinckley": "gordon-hinckley",
	"President Hinckley":           "gordon-hinckley",
	"Thomas S. Monson":             "thomas-monson",
	"President Thomas S. Monson":   "thomas-monson",
	"President Monson":             "thomas-monson",
	"Howard W. Hunter":             "howard-hunter",
	"President Howard W. Hunter":   "howard-hunter",
	"President Hunter":             "howard-hunter",
	"Ezra Taft Benson":             "ezra-benson",
	"President Ezra Taft Benson":   "ezra-benson",
	"President Benson":             "ezra-benson",
	"Spencer W. Kimball":           "spencer-kimball",
	"President Spencer W. Kimball": "spencer-kimball",
	"President Kimball":            "spencer-kimball",
	"Joseph Smith":                 "joseph-smith",
	"Prophet Joseph Smith":         "joseph-smith",
	"Joseph Smith Jr.":             "joseph-smith",

	// Relief Society General Presidents
	"Camille N. Johnson":        "camille-johnson",
	"Sister Camille N. Johnson": "camille-johnson",
	"Jean B. Bingham":           "jean-bingham",
	"Sister Jean B. Bingham":    "jean-bingham",

	// Young Women General Presidents
	"Emily Belle Freeman":        "emily-freeman",
	"Sister Emily Belle Freeman": "emily-freeman",

	// Primary General Presidents
	"Susan H. Porter":        "susan-porter",
	"Sister Susan H. Porter": "susan-porter",

	// Presiding Bishopric
	"Gerald Causse":                 "gerald-causse",
	"Bishop Gerald Causse":          "gerald-causse",
	"W. Christopher Waddell":        "christopher-waddell",
	"Bishop W. Christopher Waddell": "christopher-waddell",
	"L. Todd Budge":                 "todd-budge",
	"Bishop L. Todd Budge":          "todd-budge",
}

// lookupSpeakerHeadshot returns the headshot URL for a speaker name.
// It tries exact match first, then partial matching for common variations.
func lookupSpeakerHeadshot(name string) string {
	// Clean up the name
	name = strings.TrimSpace(name)

	// Try exact match first
	if slug, ok := speakerHeadshots[name]; ok {
		return assetsBaseURL + "/headshots/" + slug + "-square.webp"
	}

	// Try case-insensitive match
	nameLower := strings.ToLower(name)
	for key, slug := range speakerHeadshots {
		if strings.ToLower(key) == nameLower {
			return assetsBaseURL + "/headshots/" + slug + "-square.webp"
		}
	}

	// Try partial match - check if name contains a known speaker's full name
	for key, slug := range speakerHeadshots {
		// Skip short keys (Elder Oaks, etc.) for contains matching to avoid false positives
		if len(key) > 15 && strings.Contains(nameLower, strings.ToLower(key)) {
			return assetsBaseURL + "/headshots/" + slug + "-square.webp"
		}
	}

	// No match found
	return ""
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	// MCP Toolbox is required
	toolboxURL := os.Getenv("TOOLBOX_URL")
	if toolboxURL == "" {
		log.Fatal("TOOLBOX_URL environment variable is required")
	}

	// Create prophet agent with direct Gemini REST API
	log.Println("Starting with Gemini REST API and MCP Toolbox")
	prophetAgent, err := prophetagent.New(ctx, prophetagent.Config{
		ToolboxURL: toolboxURL,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	log.Println("Prophet agent initialized (Gemini REST API)")

	// Create GoFr app
	gofrApp := gofr.New()

	// Home page handler
	gofrApp.GET("/", func(ctx *gofr.Context) (interface{}, error) {
		return response.Template{
			Name: "home.html",
			Data: map[string]interface{}{
				"Title":       "What Would You Ask a Prophet?",
				"Description": "Prophets receive revelation from God and share His will for our day. They testify of Jesus Christ, warn of spiritual dangers, and make timeless truths relevant now.",
				"AssetsURL":   getEnv("ASSETS_BASE_URL", "https://storage.googleapis.com/temple-square-assets"),
			},
		}, nil
	})

	// Custom streaming endpoint that wraps agent
	// Returns HTML fragments for HTMX consumption
	gofrApp.POST("/ask", func(ctx *gofr.Context) (interface{}, error) {
		// This endpoint handles the initial POST and returns HTML for HTMX
		// GoFr Bind() supports both JSON and form-urlencoded
		var req struct {
			Question string `json:"question" form:"question"`
		}

		if err := ctx.Bind(&req); err != nil {
			return nil, fmt.Errorf("failed to parse request: %w", err)
		}

		if req.Question == "" {
			return nil, fmt.Errorf("question is required")
		}

		question := req.Question

		// Validate and classify content
		classification := prophetagent.ClassifyContent(question)
		if classification != prophetagent.ContentSafe {
			redirect := prophetagent.GetRedirectResponse(classification)
			// Render the RedirectResponse templ component as HTML
			var buf bytes.Buffer
			err := components.RedirectResponse(redirect.Message, redirect.SuggestedQuestions).Render(ctx.Request.Context(), &buf)
			if err != nil {
				return nil, fmt.Errorf("failed to render redirect response: %w", err)
			}
			// Use File response type to return raw HTML without JSON encoding
			return response.File{
				Content:     buf.Bytes(),
				ContentType: "text/html; charset=utf-8",
			}, nil
		}

		// Generate session ID for this question
		sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())

		// Render the StreamContainer templ component as HTML
		var buf bytes.Buffer
		err := components.StreamContainer(components.StreamContainerProps{
			SessionID: sessionID,
			Question:  question,
		}).Render(ctx.Request.Context(), &buf)
		if err != nil {
			return nil, fmt.Errorf("failed to render stream container: %w", err)
		}

		// Use File response type to return raw HTML without JSON encoding
		return response.File{
			Content:     buf.Bytes(),
			ContentType: "text/html; charset=utf-8",
		}, nil
	})

	// Start servers
	apiPort := getEnv("API_PORT", "8081")

	// Start internal SSE server on a separate port for SSE streaming
	// This server handles SSE requests with direct http.ResponseWriter access
	go func() {
		sseMux := http.NewServeMux()
		sseMux.HandleFunc("/api/stream", func(w http.ResponseWriter, r *http.Request) {
			handleSSEStream(w, r, prophetAgent)
		})

		// Test endpoint to debug Gemini API latency from Cloud Run
		sseMux.HandleFunc("/api/test-gemini", func(w http.ResponseWriter, r *http.Request) {
			handleTestGemini(w, r)
		})

		log.Printf("Internal SSE server starting on port %s", apiPort)
		if err := http.ListenAndServe(":"+apiPort, sseMux); err != nil {
			log.Fatalf("Internal SSE server failed: %v", err)
		}
	}()

	// Wait for internal SSE server to be ready
	time.Sleep(100 * time.Millisecond)

	// Test endpoint to debug Gemini API latency from Cloud Run
	gofrApp.GET("/api/test-gemini", func(ctx *gofr.Context) (interface{}, error) {
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return map[string]string{"error": "GEMINI_API_KEY not set"}, nil
		}

		// Test with large payload + structured output like actual format calls
		large := ctx.Param("large") == "1"
		var reqBody string
		if large {
			// Simulate ~30KB payload with structured output
			content := strings.Repeat(`{"speaker":"Test","title":"Test Talk","content":"Lorem ipsum dolor sit amet. "}`, 200)
			reqBody = fmt.Sprintf(`{
				"contents": [{"parts": [{"text": "Search results: [%s]\nSelect 2 quotes."}], "role": "user"}],
				"generationConfig": {
					"maxOutputTokens": 2048,
					"responseMimeType": "application/json",
					"responseSchema": {"type":"object","properties":{"quotes":{"type":"array"}},"required":["quotes"]}
				}
			}`, content)
		} else {
			reqBody = `{"contents": [{"parts": [{"text": "Say hello"}], "role": "user"}], "generationConfig": {"maxOutputTokens": 50}}`
		}

		start := time.Now()
		client := &http.Client{Timeout: 120 * time.Second}
		req, _ := http.NewRequest("POST",
			"https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview:generateContent",
			strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-goog-api-key", apiKey)

		resp, err := client.Do(req)
		elapsed := time.Since(start)

		if err != nil {
			return map[string]interface{}{"error": err.Error(), "elapsed_ms": elapsed.Milliseconds(), "payload_size": len(reqBody)}, nil
		}
		defer resp.Body.Close()

		return map[string]interface{}{"status": resp.StatusCode, "elapsed_ms": elapsed.Milliseconds(), "payload_size": len(reqBody)}, nil
	})

	// Create reverse proxy to forward /api/stream requests to internal SSE server
	// This allows SSE to work through GoFr by proxying to a server with raw http.ResponseWriter
	sseProxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%s", apiPort))
	sseProxy := createSSEProxy(sseProxyURL)

	// Add middleware to proxy /api/stream requests to internal SSE server
	gofrApp.UseMiddleware(sseProxyMiddleware(sseProxy))

	// Start GoFr server (SSE requests are proxied to internal server)
	log.Printf("GoFr server starting (SSE streaming proxied via /api/stream)")
	gofrApp.Run()
}

// createSSEProxy creates a reverse proxy configured for SSE streaming.
// It disables buffering and flushes responses immediately using FlushInterval.
func createSSEProxy(target *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Customize the transport to disable compression (required for SSE)
	proxy.Transport = &http.Transport{
		DisableCompression: true,
	}

	// FlushInterval of -1 means flush immediately after each write
	// This is critical for SSE streaming to work properly
	proxy.FlushInterval = -1

	// Customize the response handling to preserve SSE headers
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Ensure SSE headers are preserved
		if resp.Header.Get("Content-Type") == "text/event-stream" {
			resp.Header.Set("Cache-Control", "no-cache")
			resp.Header.Set("Connection", "keep-alive")
			resp.Header.Set("X-Accel-Buffering", "no")
		}
		return nil
	}

	return proxy
}

// flushingResponseWriter wraps http.ResponseWriter to flush after every write.
// This ensures SSE events are sent immediately to the client.
type flushingResponseWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func (frw *flushingResponseWriter) Header() http.Header {
	return frw.w.Header()
}

func (frw *flushingResponseWriter) Write(b []byte) (int, error) {
	n, err := frw.w.Write(b)
	if frw.flusher != nil {
		frw.flusher.Flush()
	}
	return n, err
}

func (frw *flushingResponseWriter) WriteHeader(statusCode int) {
	frw.w.WriteHeader(statusCode)
}

// Flush implements http.Flusher
func (frw *flushingResponseWriter) Flush() {
	if frw.flusher != nil {
		frw.flusher.Flush()
	}
}

// sseProxyMiddleware creates middleware that proxies /api/stream requests
// to the internal SSE server, enabling SSE streaming through GoFr.
func sseProxyMiddleware(proxy *httputil.ReverseProxy) gofrHTTP.Middleware {
	return func(inner http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if this is an SSE stream request
			if r.URL.Path == "/api/stream" {
				// Try to get the underlying flusher by unwrapping response writer layers
				flusher := extractFlusher(w)

				// Set SSE headers before proxying
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("X-Accel-Buffering", "no")

				// Create a flushing writer that wraps the response
				frw := &flushingResponseWriter{w: w, flusher: flusher}

				// Proxy the request to the internal SSE server
				proxy.ServeHTTP(frw, r)
				return
			}
			// Pass through to GoFr for all other requests
			inner.ServeHTTP(w, r)
		})
	}
}

// extractFlusher attempts to extract http.Flusher from a ResponseWriter.
// It handles wrapped response writers by checking if the underlying writer implements Flusher.
func extractFlusher(w http.ResponseWriter) http.Flusher {
	// Direct check
	if flusher, ok := w.(http.Flusher); ok {
		return flusher
	}

	// Try to unwrap using ResponseWriter interface pattern
	type unwrapper interface {
		Unwrap() http.ResponseWriter
	}
	if uw, ok := w.(unwrapper); ok {
		return extractFlusher(uw.Unwrap())
	}

	return nil
}

// escapeSSEData escapes data for SSE format (newlines must be prefixed with "data: ")
func escapeSSEData(s string) string {
	// SSE data fields cannot contain bare newlines - each line needs "data: " prefix
	// For single-line HTML, just return as-is
	// For multi-line, we need to split and rejoin
	lines := strings.Split(s, "\n")
	if len(lines) == 1 {
		return s
	}
	// For multi-line content, join with proper SSE continuation
	return strings.Join(lines, "\ndata: ")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
