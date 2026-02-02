# Stage 1: Build frontend
FROM node:20-alpine AS frontend

WORKDIR /app

# Copy web package files
COPY web/package.json web/pnpm-lock.yaml ./

# Install dependencies
RUN corepack enable && corepack prepare pnpm@latest --activate && \
    pnpm install --frozen-lockfile

# Copy source and build
COPY web/ .
RUN pnpm run build

# Stage 2: Build Go backend
FROM golang:1.24-alpine AS backend

WORKDIR /app

# Copy web output to static
COPY --from=frontend /app/out /app/static/out

# Copy Go source
COPY go.mod go.sum ./
COPY main.go ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY scripts/ ./scripts/
COPY static/ ./static/

# Build Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o octopus .

# Stage 3: Final runtime image
FROM alpine:3.20

ENV TZ=Asia/Shanghai

# Install runtime dependencies
RUN apk add --no-cache ca-certificates su-exec && \
    ln -fs /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    rm -rf /var/cache/apk/*

WORKDIR /app

# Copy binary and static files
COPY --from=backend /app/octopus /app/octopus
COPY --from=backend /app/static /app/static
COPY --from=backend /app/scripts /app/scripts

# Create data directory for SQLite
RUN mkdir -p /app/data

# Use su-exec for non-root user
RUN addgroup -g 1000 app && adduser -u 1000 -G app -s /bin/sh -D app

USER app

EXPOSE 8080

ENTRYPOINT ["/app/octopus"]
CMD ["start"]
