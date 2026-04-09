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

RUN apk update && apk add --no-cache ca-certificates tzdata && update-ca-certificates

# Default uploads directory should be created here
WORKDIR /app
USER appuser:appuser

COPY --from=builder /go/bin/sink /go/bin/sink

USER ${USER}:${USER}
EXPOSE 8080
ENTRYPOINT ["/go/bin/sink"]
