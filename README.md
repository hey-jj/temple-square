# Temple Square (Ask a Prophet)

## Overview
Ask-a-Prophet is a Cloud Run app that generates church quotes and scriptures using the Gemini REST API and MCP Toolbox.

## Local Development
```bash
# Start database + MCP Toolbox
make toolbox-up

# Run the app
make dev
```

Required env:
```bash
export TOOLBOX_URL=http://localhost:8082
export GEMINI_API_KEY=...
```

## Deployment

### Branch + Domain Mapping
- `main` -> Cloud Run `ask-a-prophet` -> **search.bcclab.dev**
- `concise` -> Cloud Run `deep-search` -> **app.templesquare.dev**

Two Cloudflare workers are used:
- `cloudflare-worker/` (name: `temple-square`) routes **search.bcclab.dev**
- `cloudflare-worker-app/` (name: `temple-square-app`) routes **app.templesquare.dev**

Basic auth is enabled on both workers:
- search.bcclab.dev password: `bcc-labs-25`
- app.templesquare.dev password: `temple-square`

### Cloud Run (ask-a-prophet)
```bash
gcloud run deploy ask-a-prophet \
  --source app \
  --region us-west1 \
  --project temple-square \
  --allow-unauthenticated \
  --set-env-vars="TOOLBOX_URL=https://prophet-toolbox-3izw7vdi5a-uw.a.run.app,API_PORT=8081" \
  --set-secrets="GEMINI_API_KEY=GEMINI_API_KEY:latest"
```

Verify revision:
```bash
gcloud run services describe ask-a-prophet \
  --region us-west1 \
  --project temple-square \
  --format='value(status.latestReadyRevisionName)'
```

### Cloud Run (prophet-toolbox)
```bash
gcloud run deploy prophet-toolbox \
  --source app \
  --dockerfile app/Dockerfile.toolbox \
  --region us-west1 \
  --project temple-square \
  --set-env-vars="DB_HOST=/cloudsql/temple-square:us-west1:temple-square-db,DB_NAME=conference,DB_USER=postgres,DB_SSL_MODE=disable" \
  --set-secrets="DB_PASSWORD=temple-square-db-password:latest" \
  --add-cloudsql-instances=temple-square:us-west1:temple-square-db
```

### Cloudflare Worker (search.bcclab.dev)
```bash
cd cloudflare-worker
npx wrangler secret put BASIC_AUTH_PASSWORD
npx wrangler deploy
```

### Cloudflare Worker (app.templesquare.dev)
```bash
cd cloudflare-worker-app
npx wrangler secret put BASIC_AUTH_PASSWORD
npx wrangler deploy
```

Make sure each worker's `wrangler.toml` points `BACKEND_URL` at the correct Cloud Run URL.
