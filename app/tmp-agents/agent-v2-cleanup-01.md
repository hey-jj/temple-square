# L8 Principal Engineer Handoff: V2 Clean Cutover

## Role
You are an L8 principal engineer executing a clean cutover from v1 to v2 architecture. No fallbacks, no backwards compatibility.

## Goal
Remove all v1 code paths and fallback logic from main.go, making v2 the only execution path.

## Hard Requirements
- NO conditional v1/v2 branching - v2 is the only path
- NO USE_V2 environment variable checks
- NO backwards compatibility shims
- FAIL-FAST on missing configuration (TOOLBOX_URL required)
- Remove all dead code paths

## Tasks
1. Read `/Users/justinjones/Developer/temple-square/app/cmd/server/main.go`
2. Remove the `useV2` variable and conditional branching
3. Remove the v1 agent creation path (prophetagent.New with DB)
4. Remove the v1 SSE handler selection in the /api/stream handler
5. Make TOOLBOX_URL a required environment variable (fail-fast if missing)
6. Remove the database initialization code (v2 uses MCP Toolbox, not direct DB)
7. Remove the `db` import if no longer needed
8. Ensure the server fails immediately if required config is missing

## Validation
- `go build ./cmd/server` must succeed
- No references to `USE_V2` remain
- No references to `prophetagent.New` (v1) remain
- No conditional `if useV2` logic remains
- TOOLBOX_URL missing causes immediate startup failure

## Acceptance Criteria
- Single code path using v2 architecture
- Fail-fast on missing required configuration
- No dead code or unused imports
- Clean compilation

## Report
Write your completion report to: `/Users/justinjones/Developer/temple-square/app/tmp-agents/agent-v2-cleanup-01-report.md`

Include:
- Files modified
- Lines removed
- Validation results (build output)
- Any blockers encountered

## Conventional Commit
```
refactor(server): remove v1 fallback, clean cutover to v2 architecture

BREAKING CHANGE: v1 architecture removed, TOOLBOX_URL now required
```

## Go/No-Go
No-Go if: compilation fails, v1 references remain, or fallback logic exists
