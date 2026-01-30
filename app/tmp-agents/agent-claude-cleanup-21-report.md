# CLAUDE.md Cleanup Report

## Task Summary
Cleaned up `/Users/justinjones/Developer/temple-square/CLAUDE.md` to remove v2 references and standardize architecture documentation.

## Changes Made

### 1. Renamed Section
- **Before**: "V2 Architecture (Current)"
- **After**: "Architecture"
- Removed the "(Current)" and "v2" designation since this is the only architecture

### 2. Updated Key Files References
- `internal/agent/agent_v2.go` -> `internal/agent/agent.go`
- `cmd/server/sse_v2.go` -> `cmd/server/sse.go`
- Kept `tools.yaml` and `Dockerfile.toolbox` unchanged (no v2 suffix)

### 3. Removed V2-Specific Environment Variables
- Removed `USE_V2=true` from the environment variables section
- Kept required variables: `TOOLBOX_URL`, `GOOGLE_CLOUD_PROJECT`, `GOOGLE_CLOUD_LOCATION`

### 4. Updated Local Development Commands
- **Before**: `make dev-v2`
- **After**: `make dev`

### 5. Pinned Defaults Section
- Confirmed "Pinned Defaults (MANDATORY)" section is already at the very top of the file
- Contains correct values:
  - Model: `gemini-3.0-flash-preview`
  - Reference: `https://aistackregistry.com`

## Verification
- The section header change from "V2 Architecture (Current)" to "Architecture" is complete
- All references to `_v2` in filenames have been updated
- The `USE_V2` environment variable has been removed
- The `make dev-v2` command has been changed to `make dev`

## File Location
`/Users/justinjones/Developer/temple-square/CLAUDE.md`
