FROM golang:1.20.10-alpine AS builder

RUN adduser -D -g '' questspace

WORKDIR /app/

COPY src/go.mod src/go.sum ./

RUN go mod download && \
    go mod verify

COPY src .

RUN GOOS=linux go build -o /go/bin/questspace ./cmd/questspace/main.go

FROM alpine:3.17.3
LABEL language="golang"

COPY --from=builder /etc/passwd /etc/passwd

COPY --from=builder --chown=questspace:1000 /go/bin/questspace /questspace

COPY conf /conf

COPY .env .

USER questspace

EXPOSE 8080

ENTRYPOINT ["./questspace", "--environment=docker-dev", "--config=/conf/"]
