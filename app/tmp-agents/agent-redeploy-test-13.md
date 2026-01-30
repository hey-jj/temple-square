# L8 Principal Engineer Handoff: Redeploy and Verify Fix

## Role
You are an L8 principal engineer redeploying after bug fix and verifying scriptures work.

## Context
- Bug fixed: related_talk made optional in scriptures schema
- Need to redeploy and verify scriptures section now appears

## Tasks

### 1. Verify Build
```bash
cd /Users/justinjones/Developer/temple-square/app && go build ./cmd/server
```

### 2. Redeploy Main App
```bash
cd /Users/justinjones/Developer/temple-square/app && gcloud run deploy prophet-agent \
  --source . \
  --region us-central1 \
  --project temple-square \
  --allow-unauthenticated \
  --set-env-vars="TOOLBOX_URL=https://prophet-toolbox-594677951902.us-central1.run.app,GOOGLE_CLOUD_PROJECT=temple-square,GOOGLE_CLOUD_LOCATION=us-central1,HTTP_PORT=8080"
```

### 3. Wait for Deployment
Wait 30 seconds for deployment to stabilize.

### 4. Test SSE Stream
```bash
curl -N --max-time 120 "https://prophet-agent-594677951902.us-central1.run.app/api/stream?session=test-fix-$(date +%s)&q=What%20does%20the%20Bible%20say%20about%20love%3F" 2>&1
```

### 5. Verify Scriptures Section Appears
Look for:
- `event: scriptures` in the output
- At least 5 scripture references
- Actual verse text

### 6. Check Logs
```bash
gcloud run services logs read prophet-agent --region=us-central1 --project=temple-square --limit=30 2>&1 | grep -i "scriptures\|Streamed"
```

Should see "Streamed scriptures section" in the logs.

## Report
Write to: `/Users/justinjones/Developer/temple-square/app/tmp-agents/agent-redeploy-test-13-report.md`

Include:
- Deployment status
- SSE test results
- Whether scriptures section now appears
- Go/No-Go

## Success Criteria
- Scriptures section appears in SSE output
- Contains at least 5 scripture references
- No errors
