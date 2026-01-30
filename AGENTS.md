# Agent Notes

## Deployment (current)
- Cloud Run region: `us-west1`
- Main service: `ask-a-prophet`
- Toolbox service: `prophet-toolbox`

### Deploy main app
```bash
gcloud run deploy ask-a-prophet \
  --source app \
  --region us-west1 \
  --project temple-square \
  --allow-unauthenticated \
  --set-env-vars="TOOLBOX_URL=https://prophet-toolbox-3izw7vdi5a-uw.a.run.app,API_PORT=8081" \
  --set-secrets="GEMINI_API_KEY=GEMINI_API_KEY:latest"
```

### Deploy toolbox
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

### Deploy Cloudflare Worker
```bash
cd cloudflare-worker
npx wrangler deploy
```

Update `cloudflare-worker/wrangler.toml` if the Cloud Run URL changes.
