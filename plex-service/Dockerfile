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
RUN CGO_ENABLED=0 GOOS=linux go build -o plex-service ./plex-service/main.go

# Final stage
FROM alpine:3.21

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/plex-service/main plex-service

# Create a non-root user
RUN adduser -D -g '' appuser
USER appuser

# Set non-sensitive environment variables
ENV SERVICE_PORT="50052"
ENV PLEX_HOST=""
ENV PLEX_PORT=""
ENV PLEX_CATEGORY_FILMS=""
ENV PLEX_CATEGORY_SERIES=""
ENV PLEX_CATEGORY_CARTOONS=""
ENV PLEX_CATEGORY_CARTOONS_SERIES=""
ENV PLEX_CATEGORY_SHORTS=""

# Run the application
CMD ["./plex-service"] 