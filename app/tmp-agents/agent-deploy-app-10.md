# L8 Principal Engineer Handoff: Deploy Main App

## Role
You are an L8 principal engineer deploying the prophet-agent main app to Cloud Run.

## Goal
Deploy the main app with v2 architecture connected to the MCP Toolbox server.

## Context
- MCP Toolbox URL: https://prophet-toolbox-594677951902.us-central1.run.app
- Project: temple-square
- Region: us-central1

## Tasks

### 1. Deploy Main App
```bash
cd /Users/justinjones/Developer/temple-square/app && gcloud run deploy prophet-agent \
  --source . \
  --region us-central1 \
  --project temple-square \
  --allow-unauthenticated \
  --set-env-vars="TOOLBOX_URL=https://prophet-toolbox-594677951902.us-central1.run.app,GOOGLE_CLOUD_PROJECT=temple-square,GOOGLE_CLOUD_LOCATION=us-central1,HTTP_PORT=8080"
```

### 2. Get Service URL
```bash
gcloud run services describe prophet-agent --region=us-central1 --project=temple-square --format="value(status.url)" 2>&1
```

### 3. Verify Health
Test the home page loads:
```bash
curl -s <SERVICE_URL> | head -50
```

### 4. Test Question Endpoint
Test the /ask endpoint with a simple question (POST):
```bash
curl -X POST <SERVICE_URL>/ask -d "question=What is faith?" -H "Content-Type: application/x-www-form-urlencoded" 2>&1 | head -100
```

## Report
Write to: `/Users/justinjones/Developer/temple-square/app/tmp-agents/agent-deploy-app-10-report.md`

Include:
- Deployed service URL
- Health check results
- Any errors encountered
