# Builder stage: build a small, static Go binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy Go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

ENV CGO_ENABLED=0
RUN go build -ldflags="-s -w" -trimpath -o dosync .
# Optionally, run 'upx --best --lzma dosync' here for further compression

# Create final minimal image
FROM alpine:latest

# Install Docker CLI for container control
RUN apk add --no-cache docker-cli

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/dosync .

# Make the binary executable
RUN chmod +x dosync

# Create directory for backups
RUN mkdir -p /app/backups && chmod 777 /app/backups

# We can't use a non-root user because we need access to the Docker socket
# and to modify the docker-compose.yml file

ENTRYPOINT ["./dosync"]
CMD ["sync", "-f", "/app/docker-compose.yml", "-i", "${CHECK_INTERVAL:-1m}", "${VERBOSE:-}"] 