# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies (libstdc++ required for Tailwind)
RUN apk add --no-cache git ca-certificates libstdc++ libgcc

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@v0.3.977

# Download Tailwind CLI (musl version for Alpine)
RUN wget -q https://github.com/tailwindlabs/tailwindcss/releases/download/v4.1.18/tailwindcss-linux-x64-musl -O /usr/local/bin/tailwindcss \
    && chmod +x /usr/local/bin/tailwindcss

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Generate templ
RUN templ generate

# Build Tailwind CSS
RUN tailwindcss -i static/css/input.css -o static/css/output.css -m

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# Runtime stage
FROM gcr.io/distroless/static-debian12

WORKDIR /app

COPY --from=builder /server /app/server
COPY --from=builder /app/static /app/static
COPY --from=builder /app/templates /app/templates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080

ENTRYPOINT ["/app/server"]
