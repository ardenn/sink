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

WORKDIR /app
COPY --from=builder /go/bin/sink /go/bin/sink

ENTRYPOINT ["/go/bin/sink"]
