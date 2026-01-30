# Performance Investigation & Findings

## Date: 2026-01-30

## Architecture
```
User → Cloud Run (prophet-agent, us-west1) → MCP Toolbox (prophet-toolbox, us-west1) → Cloud SQL (us-west1)
                                           → Gemini API (gemini-3-flash-preview)
```

## Critical: ALL Services in us-west1

**Database is in us-west1. All services MUST be deployed to us-west1. No exceptions.**

| Service | Region | URL |
|---------|--------|-----|
| Cloud SQL Database | us-west1 | temple-square-db |
| prophet-toolbox | us-west1 | https://prophet-toolbox-3izw7vdi5a-uw.a.run.app |
| prophet-agent | us-west1 | https://prophet-agent-3izw7vdi5a-uw.a.run.app |

### What Happens with Cross-Region

When Cloud Run/Toolbox were in us-central1 but database in us-west1:
- Tool calls: **30-40 seconds** (should be <1s)
- Total request time: **50+ seconds**

When everything is in us-west1:
- Tool calls: **<1s** historically, but currently **10–40s+** (see new findings)
- Total request time: **variable** (Gemini fast, toolbox is now the bottleneck)

## Service URLs

- **prophet-agent (us-west1)**: https://prophet-agent-594677951902.us-west1.run.app
- **prophet-toolbox (us-west1)**: https://prophet-toolbox-594677951902.us-west1.run.app
- **Note:** Cloud Run also reports hashed service URLs (e.g. `...-3izw7vdi5a-uw.a.run.app`). These are the same services.

## Current Cloud Run Config (overbuilt for debugging)

### prophet-agent (2026-01-30)
- CPU: **4 vCPU**
- Memory: **2 GiB**
- Min instances: **2**
- Max instances: **50**
- CPU throttling: **disabled** (CPU always allocated)
- Container concurrency: **10**

### prophet-toolbox (2026-01-30)
- CPU: **4 vCPU**
- Memory: **2 GiB**
- Min instances: **2**
- Max instances: **50**
- CPU throttling: **disabled** (CPU always allocated)
- Container concurrency: **80**

## New Findings (2026-01-30)

### 1) Toolbox latency is the primary bottleneck (not Gemini)
Cloud Run logs for prophet-toolbox show **30–40s+** elapsed time for multiple tools:
- `search_talks_by_speaker` ~38–39s
- `get_presidents_talks` ~39–40s
- `search_scriptures` ~36s

This indicates latency is **inside the toolbox service or DB**, not in the Gemini call or the agent formatter.

### 2) Oaks/Nelson slowness is not query-specific
Both Oaks and Nelson speaker queries show the same 30–60s latency. This points to **service-wide contention** (CPU, connection pool, or DB) rather than the query itself.

### 3) Post-scale spot check (2026-01-30 00:07Z)
- Direct toolbox call (`search_talks_by_speaker`, Oaks) from CLI: **~7.0s** total
- Cloud SQL metrics:
  - CPU utilization: **~0.50**
  - Memory utilization: **~0.45**
  - PostgreSQL backends: **2**
These are **not** showing DB saturation at the time of measurement.

### 4) Pool parameter fix (2026-01-30 00:20Z)
Applied connection pool params via `DB_NAME` for Cloud SQL source:
`pool_max_conns=50`, `pool_min_conns=10`, `pool_min_idle_conns=10`

Results (Oaks `search_talks_by_speaker`):
- **Sequential 10**: avg **1.93s**, median **1.34s**, min **0.51s**, max **7.76s**
- **Parallel 10**: avg **1.53s**, median **1.25s**, min **1.21s**, max **2.24s**

This indicates the prior 20–30s parallel latency was **pool starvation** inside toolbox.

### 5) Database oversize + higher pool (2026-01-30 00:39Z)
Cloud SQL scaled up for headroom:
- Tier: **db-custom-4-15360** (4 vCPU / 15 GB)
- Disk: **50 GB**

Toolbox pool parameters updated:
- `pool_max_conns=100`
- `pool_min_conns=5`
- `pool_min_idle_conns=5`

Parallel 10 (Oaks search) after update:
- **0.20–0.49s** per call (order-of-magnitude improvement)

## Deployment Commands

### Deploy Toolbox to us-west1
```bash
# Build the image first
gcloud builds submit --config cloudbuild-toolbox.yaml --region us-west1

# Deploy
gcloud run deploy prophet-toolbox \
  --image gcr.io/temple-square/prophet-toolbox:latest \
  --region us-west1 \
  --set-env-vars=DB_NAME=conference,DB_USER=postgres \
  --set-secrets=DB_PASSWORD=DB_PASSWORD:latest \
  --allow-unauthenticated
```

### Deploy prophet-agent to us-west1
```bash
gcloud run deploy prophet-agent \
  --source . \
  --region us-west1 \
  --set-env-vars=TOOLBOX_URL=https://prophet-toolbox-3izw7vdi5a-uw.a.run.app,HTTP_PORT=8080 \
  --set-secrets=GEMINI_API_KEY=GEMINI_API_KEY:latest \
  --allow-unauthenticated
```

## Known Issues

### 1. Gemini API Latency from Cloud Run
- **CLI tests**: 5-7 seconds per request
- **Cloud Run**: 15-25 seconds per request
- **Root cause**: Unknown - possibly rate limiting, network path, or Cloud Run egress throttling
- **Status**: Under investigation

### 2. Toolbox latency regression (current blocker)
- **Cloud Run → toolbox calls**: 10–40s+ per tool
- **Observed in toolbox logs**: per-request elapsed time in the 30–40s range
- **Status**: Active investigation (likely CPU/connection pool/DB contention)

### 2. Structured Output Performance
- Structured output (responseMimeType + responseSchema) does NOT significantly impact performance
- CLI tests confirmed similar speeds with/without structured output
- Keep using structured output for cleaner JSON responses

## Configuration

### MaxOutputTokens
- Current setting: **64000** (oversized for headroom)
- Required for complete JSON responses
- Do NOT reduce during investigation

### Model
- **gemini-3-flash-preview** (thinking model)
- DO NOT change - specified in CLAUDE.md

## CLI Testing

To test Gemini API directly:
```bash
export GEMINI_API_KEY=$(gcloud secrets versions access latest --secret=GEMINI_API_KEY --project=temple-square)

# Single request
time curl -s -X POST "https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview:generateContent?key=$GEMINI_API_KEY" \
  -H "Content-Type: application/json" \
  -d @/tmp/test_payload.json

# 6 parallel requests (simulates actual usage)
time (
  curl ... &
  curl ... &
  curl ... &
  curl ... &
  curl ... &
  curl ... &
  wait
)
```

## Lessons Learned

1. **Always check region alignment first** - Cross-region calls add massive latency
2. **Test from CLI to establish baseline** - If CLI is fast but Cloud Run is slow, the issue is Cloud Run specific
3. **Structured output is NOT the bottleneck** - Don't remove it
4. **The thinking model (gemini-3-flash-preview) takes 5-7 seconds minimum** - This is expected
5. **Tool/database calls should be <1 second** - If they're not, check region alignment
