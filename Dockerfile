# syntax=docker/dockerfile:1

FROM golang:1.25.12-alpine3.24 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/steam-profit-server ./cmd/steam-profit-server

FROM alpine:3.24.1 AS runtime

RUN apk add --no-cache ca-certificates \
    && addgroup -S steam-profit \
    && adduser -S -G steam-profit steam-profit \
    && mkdir -p /app/demo /data \
    && chown steam-profit:steam-profit /data

COPY --from=build /out/steam-profit-server /app/steam-profit-server
COPY demo/seed.csv /app/demo/seed.csv

USER steam-profit
WORKDIR /data

ENV PORT=8080 \
    STEAM_PROFIT_STORAGE=sqlite \
    STEAM_PROFIT_DATABASE_FILE=/data/steam-profit.db \
    STEAM_PROFIT_AI_PROVIDER=disabled

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -q -T 2 -O /dev/null "http://127.0.0.1:${PORT}/api/health" || exit 1

ENTRYPOINT ["/app/steam-profit-server"]
