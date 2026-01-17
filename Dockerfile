# Build stage
FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# TARGETARCH is automatically set by Docker Buildx
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -o trade-bot ./cmd/trade-bot

# Final stage
FROM alpine:latest

# Install CA certificates for secure communication with APIs
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/trade-bot .

# Expose port (if applicable, though this bot is likely outbound-only)
# EXPOSE 8080

# Default command
ENTRYPOINT ["./trade-bot"]
