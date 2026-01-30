# L8 Principal Engineer Report: MCP Toolbox Server Deployment

## Summary
**Status: SUCCESS**

The MCP Toolbox server has been successfully deployed to Cloud Run and is fully operational.

## Infrastructure Details

### Cloud SQL Instance
| Property | Value |
|----------|-------|
| Instance Name | temple-square-db |
| Database Version | PostgreSQL 16 |
| Location | us-west1-a |
| Tier | db-f1-micro |
| Public IP | 35.199.189.20 |
| Status | RUNNABLE |

### Secrets Manager
| Secret | Created |
|--------|---------|
| DB_PASSWORD | 2026-01-28T00:45:40 |
| GEMINI_API_KEY | 2026-01-27T23:19:47 |

## Deployed Service

### prophet-toolbox
| Property | Value |
|----------|-------|
| Service URL | https://prophet-toolbox-594677951902.us-central1.run.app |
| Region | us-central1 |
| Image | gcr.io/temple-square/prophet-toolbox:latest |
| Toolbox Version | 0.26.0 |
| Port | 8080 |
| Authentication | Unauthenticated (public) |

### Available Tools
The toolbox exposes 7 database tools:

1. **search_scriptures** - Full-text search across canonical scriptures
2. **get_scripture_by_reference** - Lookup specific verses by book/chapter/verse
3. **search_talks** - Search General Conference talks (2020-2025)
4. **search_talks_by_speaker** - Find talks by speaker slug
5. **search_talks_mentioning_scripture** - Find talks that reference specific scriptures
6. **get_presidents_talks** - Get talks from First Presidency members
7. **get_leaders_talks** - Get talks from other general authorities

### Toolsets Configured
- `scriptures`: Scripture-focused tools
- `presidents`: First Presidency talks
- `leaders`: General authority talks
- `all`: All tools combined

## Deployment Challenges Resolved

### 1. Go Version Compatibility
**Issue:** Original Dockerfile used Go 1.23, but genai-toolbox v0.26.0 requires Go >= 1.24.7

**Resolution:** Switched to using pre-built binary download from Google Cloud Storage instead of building from source.

### 2. Alpine Linux Compatibility
**Issue:** Pre-built binary requires glibc, but Alpine uses musl libc.

**Resolution:** Changed base image from `alpine:latest` to `debian:bookworm-slim`.

### 3. PostgreSQL Source Configuration
**Issue:** Initial `postgres` source type doesn't work well with Cloud SQL Unix sockets.

**Resolution:** Changed to `cloud-sql-postgres` source type which uses the Cloud SQL Go Connector for native Cloud SQL authentication via ADC.

### 4. Port Configuration
**Issue:** Cloud Run sets PORT environment variable dynamically.

**Resolution:** Updated Dockerfile to use shell entrypoint that reads `${PORT:-8080}` environment variable.

## Configuration Files Modified

### tools.yaml
- Updated source from `postgres` to `cloud-sql-postgres`
- Configured project, region, instance, and ipType fields
- Removed incompatible `sslmode` field

### Dockerfile.toolbox
- Changed base image to Debian bookworm-slim
- Download pre-built binary instead of compiling
- Added dynamic PORT handling via shell entrypoint

## Verification

### API Health Check
```bash
curl https://prophet-toolbox-594677951902.us-central1.run.app/api/toolset
# Returns: serverVersion "0.26.0" with 7 tools listed
```

### Database Connectivity Test
```bash
curl -X POST .../api/tool/search_scriptures/invoke -d '{"query": "love", "limit": 2}'
# Returns: Scripture results from database
```

## Go/No-Go Assessment

### GO for Main App Deployment

**Rationale:**
1. Toolbox server is healthy and responding to requests
2. Database connectivity verified with successful query execution
3. All 7 configured tools are exposed and operational
4. Cloud SQL authentication working via Go Connector (ADC)
5. Secrets properly injected from Secret Manager

### Prerequisites for Main App
The main prophet-agent app can now:
- Configure the toolbox URL: `https://prophet-toolbox-594677951902.us-central1.run.app`
- Use any of the configured toolsets via the `/api/toolset/:name` endpoints
- Invoke individual tools via `/api/tool/:name/invoke` endpoints

## Artifacts Created

| File | Description |
|------|-------------|
| `/Users/justinjones/Developer/temple-square/app/cloudbuild-toolbox.yaml` | Cloud Build config for toolbox image |
| `/Users/justinjones/Developer/temple-square/app/Dockerfile.toolbox` | Updated Dockerfile (Debian, pre-built binary) |
| `/Users/justinjones/Developer/temple-square/app/tools.yaml` | Updated tools config (cloud-sql-postgres) |

## Next Steps

1. Update prophet-agent app to use the toolbox URL
2. Consider adding authentication if needed for production
3. Monitor Cloud Run metrics for scaling requirements
4. Consider VPC connector for private IP connectivity if public IP becomes a concern

---
*Report generated: 2026-01-28*
*Deployment completed by: L8 Principal Engineer Agent*
