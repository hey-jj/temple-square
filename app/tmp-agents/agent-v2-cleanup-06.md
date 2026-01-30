# L8 Principal Engineer Handoff: Clean Build Files

## Role
You are an L8 principal engineer removing dead configuration references.

## Goal
Remove all USE_V2 references from build and configuration files.

## Hard Requirements
- No USE_V2 environment variable references
- v2 is the only architecture - no toggle needed
- Clean configuration with no dead options

## Tasks
1. Read `/Users/justinjones/Developer/temple-square/app/Makefile`
2. Remove any USE_V2 references
3. Update `dev-v2` target to just be `dev` (or remove the distinction)
4. Read `/Users/justinjones/Developer/temple-square/app/docker-compose.yaml`
5. Remove any USE_V2 references (even in comments)
6. Ensure TOOLBOX_URL is properly configured (required, not optional)

## Validation
- `grep -r "USE_V2" .` returns no results in Makefile or docker-compose.yaml
- Makefile has clean targets without v1/v2 distinction

## Report
Write completion report to: `/Users/justinjones/Developer/temple-square/app/tmp-agents/agent-v2-cleanup-06-report.md`

## Conventional Commit
```
chore(build): remove USE_V2 references, v2 is only architecture
```
