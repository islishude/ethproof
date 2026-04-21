# syntax=docker/dockerfile:1
FROM golang:1.26.2 AS compiler
WORKDIR /app
COPY . ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o gotgz -trimpath -ldflags="-s -w" ./cmd/...

FROM alpine:latest AS certs
RUN apk --no-cache add ca-certificates

FROM gcr.io/distroless/base-debian13:latest
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=compiler /app/bin/ethproof /usr/local/bin/
ENTRYPOINT [ "ethproof" ]
