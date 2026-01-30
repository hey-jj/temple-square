# L8 Principal Engineer Handoff: Resolve Quota Issues

## Role
You are an L8 principal engineer resolving Vertex AI quota exhaustion.

## Context
- Vertex AI Gemini 2.0 Flash is returning 429 RESOURCE_EXHAUSTED errors
- This is blocking all testing and production use
- The application code is correct but cannot execute due to quota limits

## Options to Investigate

### Option 1: Check Quota Details
```bash
# View current quotas
gcloud alpha services quota describe --consumer=projects/temple-square --service=aiplatform.googleapis.com 2>&1

# Or via console URL
echo "https://console.cloud.google.com/iam-admin/quotas?project=temple-square&service=aiplatform.googleapis.com"
```

### Option 2: Request Quota Increase
If the quota is too low, request an increase through:
- Google Cloud Console Quotas page
- Or via gcloud

### Option 3: Switch to Different Model
Check if gemini-1.5-flash or another model has available quota:
```bash
# Update agent_v2.go to use gemini-1.5-flash instead of gemini-2.0-flash
```

### Option 4: Add Retry with Backoff
Implement exponential backoff in the agent to handle transient 429 errors.
However, this won't help if quota is truly exhausted.

### Option 5: Use Different Region
Try a different Vertex AI region that may have different quotas:
```bash
# Update GOOGLE_CLOUD_LOCATION from us-central1 to another region
```

## Tasks

### 1. Diagnose Quota Status
Determine current quota usage and limits.

### 2. Identify Best Solution
Based on diagnosis, recommend the best path forward.

### 3. Implement Fix
If switching models or regions, update the code and redeploy.

### 4. Test
Verify the fix resolves the 429 errors.

## Report
Write to: `/Users/justinjones/Developer/temple-square/app/tmp-agents/agent-quota-fix-15-report.md`

Include:
- Quota diagnosis results
- Solution implemented
- Test results
