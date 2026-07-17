FROM golang:1.25-alpine AS builder
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -u 1001 synx

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build \
    -ldflags="-w -s -X main.Version=${VERSION}" \
    -o /bin/vvm .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo                 /usr/share/zoneinfo
COPY --from=builder /etc/passwd                         /etc/passwd
COPY --from=builder /bin/vvm                            /bin/vvm

USER synx

EXPOSE 8332

ENTRYPOINT ["/bin/vvm"]