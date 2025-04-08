# Build stage
FROM golang:1.23-alpine3.21 AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod file
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o bot ./bot/main.go

# Final stage
FROM alpine:3.21

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/bot/main bot

# Create a non-root user
RUN adduser -D -g '' appuser
USER appuser

# Set non-sensitive environment variables
ENV ALLOWED_USERS=""
ENV COORDINATOR_SERVICE_URL=""

# Run the application
CMD ["./bot"] 