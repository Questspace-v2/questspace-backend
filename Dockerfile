FROM golang:1.22.1-alpine3.19 AS builder

RUN adduser -D -g '' questspace

RUN wget "https://storage.yandexcloud.net/cloud-certs/CA.pem" \
         --output-document /root.crt

WORKDIR /app/

COPY src/go.mod src/go.sum ./

RUN go mod download && \
    go mod verify

COPY src .

RUN GOOS=linux go build -o /go/bin/questspace -ldflags "-s -w" ./cmd/questspace/main.go

FROM alpine:3.19.1
LABEL language="golang"

COPY --from=builder /etc/passwd /etc/passwd

COPY --from=builder --chown=questspace:1000 /go/bin/questspace /questspace

RUN mkdir -p /home/questspace/.postgresql

COPY --from=builder --chown=questspace:1000 --chmod=0600 /root.crt /home/questspace/.postgresql/root.crt

COPY conf /conf

USER questspace

EXPOSE 8080

ENTRYPOINT ["./questspace", "--config=/conf/"]
