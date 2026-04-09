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

RUN apk add --no-cache ca-certificates tzdata && update-ca-certificates

# Create a non-root user and set up the application directory
RUN adduser -D appuser && mkdir /app && chown appuser:appuser /app
WORKDIR /app

# Default uploads directory should be created here
WORKDIR /app
USER appuser:appuser

COPY --from=builder /go/bin/sink /go/bin/sink

EXPOSE 8080
ENTRYPOINT ["/go/bin/sink"]
