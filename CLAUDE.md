## Pinned Defaults (MANDATORY)

**THIS SECTION IS NON-NEGOTIABLE. ALL AGENTS MUST FOLLOW THESE DEFAULTS.**

| Setting | Value | Notes |
|---------|-------|-------|
| **Model** | `gemini-3-flash-preview` | NEVER use 2.0, 2.5, or any other version. Always use 3-flash-preview. |
| **Backend** | Direct REST API | Do NOT use genai/ADK libraries (buggy in Cloud Run). Use raw HTTP. |
| **Region** | `us-central1` | Iowa - required for all deployments (same datacenter as future services) |
| **Reference** | https://aistackregistry.com | Authoritative source for model defaults and configurations |

### Model Specification (from aistackregistry.com)

**Spec URL**: https://aistackregistry.com/latest/models/gemini/gemini-3-flash-preview/spec.json
**Defaults URL**: https://aistackregistry.com/latest/models/gemini/gemini-3-flash-preview/recommended_defaults.json

```json
{
  "as_of": "2026-01-28T02:50:10+00:00",
  "model_id": "gemini-3-flash-preview",
  "provider": "google",
  "schema_version": "0.1.0",
  "thinking_level": "high",
  "modalities": ["text", "image", "video", "audio", "pdf"],
  "token_limits": {
    "input_token_limit": 1048576,
    "output_token_limit": 65536
  },
  "sources": [
    "https://ai.google.dev/gemini-api/docs/thinking",
    "https://ai.google.dev/gemini-api/docs/gemini-3.md.txt",
    "https://ai.google.dev/api/models",
    "https://ai.google.dev/gemini-api/docs/models/gemini",
    "https://generativelanguage.googleapis.com/v1beta/models",
    "https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview"
  ]
}
```

**CRITICAL**: The model is accessed via `generativelanguage.googleapis.com` (Gemini API), NOT Vertex AI.

**CRITICAL RULES:**
1. When specifying Gemini models, ALWAYS use `gemini-3-flash-preview`
2. Do NOT substitute with `gemini-2.0-flash`, `gemini-2.5-flash`, `gemini-3.0-flash-preview`, or any other variant
3. Use direct REST calls to `generativelanguage.googleapis.com` - DO NOT use genai/ADK libraries (they have bugs in Cloud Run)
4. Set `GEMINI_API_KEY` environment variable (from Google Secret Manager)
5. All Cloud Run deployments MUST use region `us-central1`

---

# Temple Square Project - Claude Reference

## Local Reference Repositories (Authoritative)

**IMPORTANT**: The following local repositories are the gold standard for patterns and implementations. These are maintained by the project owner and supersede LLM training data. Always prefer these sources over web documentation or training knowledge.

Agents should use Explore agents (or read directly) to find patterns, implementations, and best practices from these repositories.

| Repository | Local Path | Description |
|------------|------------|-------------|
| A2A Go SDK | `/Users/justinjones/Developer/agent-references/a2a-go` | Agent2Agent protocol - agent communication patterns |
| GenAI Toolbox | `/Users/justinjones/Developer/agent-references/genai-toolbox` | GenAI Toolbox source - database tool patterns, toolset configurations |
| Google Cloud Go SDK | `/Users/justinjones/Developer/agent-references/google-cloud-go` | Google Cloud Go SDK - Cloud services integration patterns |
| MCP Toolbox SDK Go | `/Users/justinjones/Developer/agent-references/mcp-toolbox-sdk-go` | MCP Toolbox SDK for Go - tbadk integration, tool loading |

### Usage Guidelines

1. **When implementing agent patterns**: Use A2A protocol or raw goroutines for parallel execution
2. **When integrating MCP Toolbox**: Reference `/Users/justinjones/Developer/agent-references/mcp-toolbox-sdk-go` for tbadk package usage
3. **When configuring database tools**: Check `/Users/justinjones/Developer/agent-references/genai-toolbox` for tools.yaml patterns
4. **When calling Gemini API**: Use direct REST calls (see below), NOT genai/ADK libraries

### Gemini API (Direct REST - MANDATORY)

**DO NOT USE**: `google.golang.org/adk` or `google.golang.org/genai` - these libraries have bugs that ignore BackendGeminiAPI in Cloud Run environments.

**USE**: Direct REST calls to Gemini API:
```bash
curl "https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview:generateContent" \
  -H "x-goog-api-key: $GEMINI_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{
    "contents": [{"parts": [{"text": "..."}]}],
    "generationConfig": {
        "responseMimeType": "application/json",
        "responseJsonSchema": {...}
    }
  }'
```

For streaming: use `:streamGenerateContent?alt=sse` endpoint.

**Note**: These repositories are kept updated regularly. When in doubt about an API or pattern, explore these local sources first before relying on external documentation or training data.

---

## GenAI Toolbox for Databases

### Core Documentation
- **GenAI Toolbox Repository**: https://github.com/googleapis/genai-toolbox
- **MCP Toolbox SDK Go**: https://github.com/googleapis/mcp-toolbox-sdk-go
- **MCP Toolbox for Databases**: https://google.github.io/adk-docs/mcp/#mcp-toolbox-for-databases

### Installation
```bash
go get github.com/googleapis/mcp-toolbox-sdk-go
```

### Key Features
- Universal abstraction layer for database access
- Built-in support for 40+ databases including PostgreSQL, Cloud SQL
- Production-ready tools for Gen AI agents
- Secure backend data source exposure
- Direct MCP capabilities for agents

## Gemini API (Direct REST)

### Structured Outputs
- **Documentation**: https://ai.google.dev/gemini-api/docs/structured-output

### Key Points
- Use `responseMimeType: "application/json"` in generationConfig
- Define `responseJsonSchema` with complete type specifications
- Call REST endpoint directly - DO NOT use genai/ADK Go libraries

### REST Endpoint
```
POST https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview:generateContent
Header: x-goog-api-key: $GEMINI_API_KEY
Header: Content-Type: application/json

For streaming:
POST https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview:streamGenerateContent?alt=sse
```

## Architecture Principles

### Agent Design
1. **Use workflow agents** (sequential/parallel) instead of relying on LLM tool-calling decisions
2. **Use structured outputs** for predictable, type-safe responses
3. **Keep agent scopes small** - each agent has one focused responsibility
4. **Use MCP Toolbox** for direct database access instead of custom connections

### Data Flow
1. **RAG/pgvector** for quick semantic matching
2. **MCP Toolbox** for structured data extraction
3. **Parallel agents** for concurrent section generation
4. **Structured outputs** for reliable parsing

## Architecture

### Overview
The architecture uses parallel agents with structured outputs for reliable, predictable responses.

### Key Files
- `internal/agent/agent.go` - Parallel agent definition with MCP Toolbox
- `cmd/server/sse.go` - SSE handler for structured JSON outputs
- `tools.yaml` - MCP Toolbox configuration (database tools)
- `Dockerfile.toolbox` - MCP Toolbox server deployment

### Environment Variables
```bash
TOOLBOX_URL=http://...   # MCP Toolbox server URL
GEMINI_API_KEY=...       # Gemini API key (from Google Secret Manager: gemini-api-key)
GOOGLE_CLOUD_PROJECT=... # GCP project (for Cloud Run deployment)
GOOGLE_CLOUD_LOCATION=.. # Region (us-central1)
```

### Secrets (Google Secret Manager)
```bash
GEMINI_API_KEY            # Gemini API key for generativelanguage.googleapis.com
temple-square-db-password # Database password for Cloud SQL
```

### Local Development
```bash
# Start MCP Toolbox and database
make toolbox-up

# Run app
make dev
```

### Parallel Agent Structure
```
prophet_agent (goroutines)
├── presidents_agent → Gemini REST + MCP Toolbox → JSON structured output
├── leaders_agent → Gemini REST + MCP Toolbox → JSON structured output
└── scriptures_agent → Gemini REST + MCP Toolbox → JSON structured output
```

All three sub-agents run concurrently via goroutines, each calling Gemini REST API directly with MCP Toolbox for database access.

## Deployment

### Cloud Run Services
| Service | URL | Description |
|---------|-----|-------------|
| prophet-agent | https://prophet-agent-3izw7vdi5a-uc.a.run.app | Main API server (Go) |
| prophet-toolbox | https://prophet-toolbox-3izw7vdi5a-uc.a.run.app | MCP Toolbox server |

### Cloudflare Worker (Proxy)
The Cloudflare Worker proxies requests to Cloud Run, handling SSE streaming correctly.

| URL | Status | Notes |
|-----|--------|-------|
| https://app.templesquare.dev | ✅ Working | Primary custom domain |
| https://temple-square.labs-testing.workers.dev | ✅ Working | workers.dev fallback |

**Worker Configuration**: `/cloudflare-worker/wrangler.toml`

**Deploy Worker**:
```bash
cd cloudflare-worker && npx wrangler deploy
```

### Deploy Cloud Run
```bash
cd app && gcloud run deploy prophet-agent --source . --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars="TOOLBOX_URL=https://prophet-toolbox-3izw7vdi5a-uc.a.run.app,HTTP_PORT=8080" \
  --set-secrets="GEMINI_API_KEY=GEMINI_API_KEY:latest"
```

## Project-Specific Notes

### Scripture Section Requirements
- Minimum 5 scripture outputs per response
- Each scripture includes a small pull quote from a related conference talk
- Different formatting (table/card layout) from speaker quotes
- No headshot for talk references - focus on scriptures

### Response Sections
1. Church Presidents (with headshots, substantial quotes)
2. Other Church Leaders (with headshots, substantial quotes)
3. Related Scriptures (5+ scriptures, each with related talk pull quote)

### Toolsets Configuration (tools.yaml)
```yaml
toolsets:
  scriptures:     # For scriptures_agent
    - search_scriptures
    - get_scripture_by_reference
    - search_talks_mentioning_scripture

  presidents:     # For presidents_agent
    - get_presidents_talks
    - search_talks_by_speaker

  leaders:        # For leaders_agent
    - get_leaders_talks
    - search_talks
```
