##################################
# STEP 1 build executable binary
#################################
FROM golang:1.26.0-alpine AS builder

ENV USER=appuser
ENV UID=1000

# See https://stackoverflow.com/a/55757473/12429735RUN
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

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

RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /go/bin/sink /go/bin/sink

USER ${USER}:${USER}
EXPOSE 8080
ENTRYPOINT ["/go/bin/sink"]
