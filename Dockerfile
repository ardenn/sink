##################################
# STEP 1 build executable binary
#################################
FROM golang:1.26.0-alpine AS builder

WORKDIR /src/app
COPY go.* ./

RUN go mod download
RUN go mod verify

COPY . .

# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux go build -o /go/bin/sink main.go

#############################
# STEP 2 build a small image
#############################
FROM alpine:latest

# Install dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create the user (default 1000)
RUN adduser -D -u 1000 appuser && \
    mkdir -p /app /appdata/uploads && \
    chown -R appuser:appuser /app /appdata

WORKDIR /app
COPY --from=builder /go/bin/sink /usr/local/bin/sink

# Switch to the non-root user
USER appuser

ENTRYPOINT ["/usr/local/bin/sink"]
