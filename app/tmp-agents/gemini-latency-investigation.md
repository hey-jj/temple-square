# Gemini API Latency Investigation - Cloud Run vs Local CLI

## Role
You are an L8 Google Principal Engineer specializing in distributed systems, Cloud Run performance, and API optimization. Your investigation must be systematic, evidence-driven, and produce actionable findings.

## Problem Statement
Gemini API calls from Cloud Run take 15-25 seconds, but identical calls from local CLI take 5-7 seconds. This is a 3-4x performance degradation with no obvious cause.

## What Has Been Established (DO NOT RE-INVESTIGATE)

### Confirmed NOT the issue:
1. **Structured output** - CLI tests with/without structured output show same ~6s latency
2. **MaxOutputTokens setting** - Tested at 2048, 4096, 16384 - no significant difference
3. **Database/Tool latency** - Fixed by moving to us-west1, now 37-362ms
4. **Model selection** - Must use `gemini-3-flash-preview` per requirements
5. **Payload size** - Tested 1KB to 30KB payloads, linear scaling as expected

### Current Architecture (all us-west1):
```
Cloud Run (prophet-agent) ---> Gemini API (generativelanguage.googleapis.com)
                          ---> Cloud Run (prophet-toolbox) ---> Cloud SQL
```

### Measured Latencies:
| Source | Single Request | 6 Parallel Requests |
|--------|---------------|---------------------|
| Local CLI (curl) | 5-7 seconds | 7-8 seconds total |
| Cloud Run | 15-25 seconds | 60-90+ seconds total |

### Current HTTP Client Config (gemini_client.go):
```go
httpClient: &http.Client{
    Timeout: 120 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 20,
        MaxConnsPerHost:     20,
        IdleConnTimeout:     90 * time.Second,
    },
},
```

## Investigation Areas (Evidence Required)

### 1. Cloud Run Instance Configuration
- Current CPU/Memory allocation
- Cold start impact vs warm instance
- CPU throttling during request processing
- Does increasing CPU/Memory reduce latency?

Commands to gather evidence:
```bash
gcloud run services describe prophet-agent --region us-west1 --format='yaml(spec.template.spec.containers[0].resources)'
gcloud run services describe prophet-agent --region us-west1 --format='yaml(spec.template.metadata.annotations)'
```

### 2. Network Egress Path
- Is Cloud Run egress going through a NAT/proxy?
- VPC connector impact?
- Direct egress vs serverless VPC access?
- MTU issues?
- DNS resolution time for generativelanguage.googleapis.com from Cloud Run?

### 3. Connection Reuse
- Is HTTP/2 being used? (Gemini API supports it)
- Are connections being reused or re-established per request?
- TCP connection setup overhead?
- TLS handshake overhead?

Test: Add logging to track connection reuse:
```go
Transport: &http.Transport{
    DisableKeepAlives: false, // ensure this is false
    ForceAttemptHTTP2: true,  // try HTTP/2
    // ... existing config
},
```

### 4. Request Queuing / Concurrency
- Cloud Run concurrency setting (currently 80)
- Are requests being queued at Cloud Run level?
- Gemini API rate limiting per source IP?
- Does Cloud Run share egress IP with other tenants?

### 5. CPU Allocation During Network I/O
- Cloud Run "CPU always allocated" vs "CPU only during request"
- Does the instance get CPU-throttled while waiting for Gemini response?

Check with:
```bash
gcloud run services describe prophet-agent --region us-west1 --format='yaml(spec.template.metadata.annotations["run.googleapis.com/cpu-throttling"])'
```

### 6. Gemini API Endpoint Selection
- Is there a regional Gemini endpoint that would be faster?
- Current: generativelanguage.googleapis.com (global)
- Alternative: us-west1-aiplatform.googleapis.com?

### 7. Request/Response Size Impact
- Measure TTFB (time to first byte) vs total time
- Is latency in request upload, processing, or response download?
- Add timing instrumentation to isolate phases

### 8. Comparison Test
Create a minimal Cloud Run service that ONLY makes Gemini API calls (no database, no other logic) and measure latency. This isolates whether the issue is Cloud Run → Gemini or something in our application.

## Required Deliverables

1. **Evidence table** showing each hypothesis tested with actual measurements
2. **Cloud Run configuration changes** tested with before/after metrics
3. **Recommended configuration** with expected improvement
4. **Any code changes** needed to optimize the HTTP client

## Code Locations

- HTTP Client: `/Users/justinjones/Developer/temple-square/app/internal/agent/gemini_client.go`
- Main server: `/Users/justinjones/Developer/temple-square/app/cmd/server/main.go`
- Service: prophet-agent in us-west1

## API Key
```bash
export GEMINI_API_KEY=$(gcloud secrets versions access latest --secret=GEMINI_API_KEY --project=temple-square)
```

## Success Criteria
Reduce Cloud Run → Gemini latency to within 50% of CLI baseline (target: <10 seconds per request).
