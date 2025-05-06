FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy Go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/dosync .

# Create final minimal image
FROM alpine:latest

# Install Docker CLI for container control
RUN apk add --no-cache docker-cli

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/dosync /usr/local/bin/dosync

# Make the binary executable
RUN chmod +x /usr/local/bin/dosync

# Create directory for backups
RUN mkdir -p /app/backups && chmod 777 /app/backups

# We can't use a non-root user because we need access to the Docker socket
# and to modify the docker-compose.yml file

ENTRYPOINT ["dosync"]
CMD ["sync", "-f", "/app/docker-compose.yml", "-i", "${CHECK_INTERVAL:-1m}", "${VERBOSE:-}"] 