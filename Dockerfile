# Multi-stage build for production
FROM node:18-alpine AS frontend-builder

WORKDIR /app
COPY package*.json ./
RUN npm ci --prefer-offline --no-audit

COPY static/css/input.css ./static/css/
COPY tailwind.config.js ./
RUN npx tailwindcss -i ./static/css/input.css -o ./static/css/styles.css --minify

# Go builder stage
FROM golang:1.24-alpine AS go-builder

# Install dependencies for building
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

# Copy only necessary source files
COPY cmd ./cmd
COPY internal ./internal
COPY templates ./templates

# Generate templates
RUN templ generate

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/server cmd/server/main.go

# Final stage
FROM alpine:latest

# Install necessary packages
RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -S appuser -u 1001 -G appgroup && \
    mkdir -p /app/logs && \
    chown -R appuser:appgroup /app

# Copy built application
COPY --from=go-builder /app/bin/server .
COPY --from=frontend-builder /app/static/css/styles.css ./static/css/

# Copy other static assets if they exist
COPY static ./static

# Switch to non-root user
USER appuser

# Set default PORT (can be overridden by environment)
ENV PORT=8080

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:${PORT}/health || exit 1

# Run the application
CMD ["./server"]