# Build stage
FROM golang:1.24-alpine AS builder

# Install git for go mod download
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build all CLIs
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/alpha ./cmd/alpha
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/beta ./cmd/beta
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/foundry ./cmd/foundry

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/partir ./cmd/partir
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/guard ./cmd/guard
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/migrate ./cmd/migrate

# Runtime stage
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 partir && adduser -u 1000 -G partir -s /bin/sh -D partir

# Copy binaries from builder
COPY --from=builder /bin/alpha /usr/local/bin/
COPY --from=builder /bin/beta /usr/local/bin/
COPY --from=builder /bin/foundry /usr/local/bin/

COPY --from=builder /bin/partir /usr/local/bin/
COPY --from=builder /bin/guard /usr/local/bin/
COPY --from=builder /bin/migrate /usr/local/bin/

# Copy migrations
COPY --from=builder /app/migrations /app/migrations

# Set working directory
WORKDIR /app

# Switch to non-root user
USER partir

# Default command
CMD ["partir", "version"]
