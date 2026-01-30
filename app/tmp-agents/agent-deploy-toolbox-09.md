# L8 Principal Engineer Handoff: Deploy MCP Toolbox Server

## Role
You are an L8 principal engineer deploying the MCP Toolbox server to Cloud Run.

## Goal
Deploy the MCP Toolbox server and verify it's operational.

## Context
- Project: temple-square
- Region: us-central1
- Database: Cloud SQL PostgreSQL instance exists

## Tasks

### 1. Check Current Cloud Run Services
```bash
gcloud run services list --region=us-central1 --project=temple-square 2>&1
```

### 2. Check Cloud SQL Instances
```bash
gcloud sql instances list --project=temple-square 2>&1
```

### 3. Deploy MCP Toolbox Server
Deploy using the Dockerfile.toolbox:
```bash
cd /Users/justinjones/Developer/temple-square/app && gcloud run deploy prophet-toolbox \
  --source . \
  --dockerfile Dockerfile.toolbox \
  --region us-central1 \
  --project temple-square \
  --allow-unauthenticated \
  --set-env-vars="DB_NAME=conference,DB_USER=postgres,DB_SSL_MODE=disable" \
  --add-cloudsql-instances=temple-square:us-central1:temple-square-db \
  --set-env-vars="DB_HOST=/cloudsql/temple-square:us-central1:temple-square-db"
```

Note: DB_PASSWORD should be set via Secret Manager. Check if secret exists:
```bash
gcloud secrets list --project=temple-square 2>&1
```

### 4. Verify Deployment
Get the service URL and verify it responds:
```bash
gcloud run services describe prophet-toolbox --region=us-central1 --project=temple-square --format="value(status.url)" 2>&1
```

### 5. Test Toolbox Health
If URL is available, test basic connectivity.

## Report
Write to: `/Users/justinjones/Developer/temple-square/app/tmp-agents/agent-deploy-toolbox-09-report.md`

Include:
- Cloud SQL instance details
- Deployed service URL
- Any errors encountered
- Go/No-Go for main app deployment

## Blockers
If Cloud SQL instance doesn't exist or secrets aren't configured, document what's needed but don't fail - the infrastructure may need separate setup.
