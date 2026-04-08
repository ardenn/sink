# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Install dependencies first (leverage Docker cache)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build a statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -o sink main.go

# Run stage
FROM alpine:latest

WORKDIR /app

# Create a non-root user and group
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy the binary and config from the builder
COPY --from=builder /app/sink /app/sink
COPY config.yaml /app/config.yaml

# Create the uploads directory and configure permissions
RUN mkdir -p /app/uploads && \
    chown -R appuser:appgroup /app

# Switch to the non-root user
USER appuser

# Expose the port the app runs on
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/app/sink"]
