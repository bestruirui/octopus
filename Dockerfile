# =============================================================================
# Build stage for frontend
# =============================================================================
FROM node:20-alpine AS frontend-builder

WORKDIR /build

# Install pnpm
RUN corepack enable && corepack prepare pnpm@latest --activate

# Copy frontend package files
COPY web/package.json web/pnpm-lock.yaml ./

# Install dependencies
RUN pnpm install --frozen-lockfile

# Copy frontend source
COPY web/ ./

# Build frontend with version injected
ARG APP_VERSION=dev
RUN NEXT_PUBLIC_APP_VERSION="${APP_VERSION}" pnpm build

# =============================================================================
# Build stage for Go binary
# =============================================================================
FROM golang:1.24-alpine AS go-builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy frontend build output to static directory
COPY --from=frontend-builder /build/out ./static/out

# Ensure _not-found has a placeholder file for go:embed
RUN if [ -d "static/out/_not-found" ] && [ ! -f "static/out/_not-found/.keep" ]; then \
        echo 'placeholder for go:embed' > static/out/_not-found/.keep; \
    fi

# Build arguments for version info
ARG APP_VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-X 'github.com/lingyuins/octopus/internal/conf.Version=${APP_VERSION}' \
              -X 'github.com/lingyuins/octopus/internal/conf.Commit=${GIT_COMMIT}' \
              -X 'github.com/lingyuins/octopus/internal/conf.BuildTime=${BUILD_TIME}' \
              -X 'github.com/lingyuins/octopus/internal/conf.Author=lingyu' \
              -s -w" \
    -tags=jsoniter \
    -o octopus \
    .

# =============================================================================
# Runtime stage
# =============================================================================
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 octopus && \
    adduser -u 1000 -G octopus -s /bin/sh -D octopus

WORKDIR /app

# Copy binary
COPY --from=go-builder /build/octopus .

# Create data directory
RUN mkdir -p /app/data && chown -R octopus:octopus /app

# Switch to non-root user
USER octopus

# Expose port
EXPOSE 8080

# Set default data directory
ENV OCTOPUS_DATA_DIR=/app/data

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/status || exit 1

# Run the binary
ENTRYPOINT ["./octopus"]
CMD ["start"]
