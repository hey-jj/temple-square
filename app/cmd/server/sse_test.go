// cmd/server/sse_test.go
package main

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestSSEProxyStreaming verifies that the SSE proxy correctly streams events
func TestSSEProxyStreaming(t *testing.T) {
	// Create a mock SSE backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("Backend: ResponseWriter doesn't support Flusher")
			return
		}

		// Send 3 SSE events with delays to simulate streaming
		for i := 1; i <= 3; i++ {
			fmt.Fprintf(w, "event: message\ndata: Event %d\n\n", i)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)
		}
		fmt.Fprintf(w, "event: done\ndata: \n\n")
		flusher.Flush()
	}))
	defer backend.Close()

	// Create the SSE proxy pointing to our mock backend
	backendURL, _ := url.Parse(backend.URL)
	proxy := createSSEProxy(backendURL)

	// Create a test server that uses our proxy middleware
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate what the middleware does
		flusher := extractFlusher(w)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		frw := &flushingResponseWriter{w: w, flusher: flusher}
		proxy.ServeHTTP(frw, r)
	}))
	defer testServer.Close()

	// Make request to the proxy
	resp, err := http.Get(testServer.URL + "/api/stream?session=test&q=hello")
	if err != nil {
		t.Fatalf("Failed to connect to proxy: %v", err)
	}
	defer resp.Body.Close()

	// Verify response headers
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("Expected Content-Type text/event-stream, got %s", resp.Header.Get("Content-Type"))
	}

	// Read and verify SSE events
	scanner := bufio.NewScanner(resp.Body)
	var events []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			events = append(events, line)
		}
	}

	if len(events) < 3 {
		t.Errorf("Expected at least 3 data events, got %d: %v", len(events), events)
	}

	t.Logf("Received %d SSE events: %v", len(events), events)
}

// TestExtractFlusher verifies flusher extraction works
func TestExtractFlusher(t *testing.T) {
	// Standard ResponseWriter from httptest should support Flusher
	w := httptest.NewRecorder()

	flusher := extractFlusher(w)
	if flusher == nil {
		t.Error("Expected to extract Flusher from httptest.ResponseRecorder")
	}
}

// TestFlushingResponseWriter verifies the flushing wrapper works
func TestFlushingResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	flusher := extractFlusher(w)

	frw := &flushingResponseWriter{w: w, flusher: flusher}

	// Test Header()
	frw.Header().Set("X-Test", "value")
	if w.Header().Get("X-Test") != "value" {
		t.Error("Header not set correctly")
	}

	// Test WriteHeader()
	frw.WriteHeader(http.StatusOK)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test Write()
	n, err := frw.Write([]byte("test data"))
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != 9 {
		t.Errorf("Expected 9 bytes written, got %d", n)
	}

	// Test Flush()
	frw.Flush() // Should not panic
}
