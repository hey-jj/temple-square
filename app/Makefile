.PHONY: dev build test clean tailwind templ

# Development (requires toolbox running: make toolbox-up)
# Set GEMINI_API_KEY env var or use: gcloud secrets versions access latest --secret=gemini-api-key
dev: templ tailwind
	TOOLBOX_URL=http://127.0.0.1:5000 go run ./cmd/server

# Build
build: templ tailwind
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

# Tailwind CSS
tailwind:
	./scripts/tailwindcss -i static/css/input.css -o static/css/output.css

tailwind-watch:
	./scripts/tailwindcss -i static/css/input.css -o static/css/output.css --watch

# templ
templ:
	templ generate

templ-watch:
	templ generate --watch

# Docker
docker-build:
	docker build -t prophet-agent .

docker-build-toolbox:
	docker build -f Dockerfile.toolbox -t prophet-toolbox .


# Start MCP Toolbox server locally (requires database running)
toolbox-up:
	docker-compose up -d db toolbox

toolbox-down:
	docker-compose down

# Deploy to Cloud Run
deploy:
	gcloud run deploy prophet-agent \
		--source . \
		--region us-central1 \
		--allow-unauthenticated \
		--set-env-vars="TOOLBOX_URL=https://toolbox-xxx-uc.a.run.app,GOOGLE_CLOUD_PROJECT=temple-square,GOOGLE_CLOUD_LOCATION=us-central1" \
		--set-secrets="GEMINI_API_KEY=GEMINI_API_KEY:latest"

deploy-toolbox:
	gcloud run deploy prophet-toolbox \
		--source . \
		--dockerfile Dockerfile.toolbox \
		--region us-central1 \
		--set-env-vars="DB_HOST=/cloudsql/temple-square:us-central1:temple-square-db,DB_NAME=conference,DB_USER=postgres,DB_SSL_MODE=disable" \
		--set-secrets="DB_PASSWORD=temple-square-db-password:latest" \
		--add-cloudsql-instances=temple-square:us-central1:temple-square-db

# Test
test:
	go test -v ./...

# Clean
clean:
	rm -rf bin/
	rm -f static/css/output.css
